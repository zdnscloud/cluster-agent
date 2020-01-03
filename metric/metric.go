package metric

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/prometheus/common/expfmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/helper"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/gorest/resource"

	common "github.com/zdnscloud/cluster-agent/commonresource"
)

const (
	MetricsURL                  = "http://%s:%d/%s"
	DefaultMetricPath           = "metrics"
	AnnotationsPrometheusPath   = "prometheus.io/path"
	AnnotationsPrometheusPort   = "prometheus.io/port"
	AnnotationsPrometheusScrape = "prometheus.io/scrape"
)

type Workloads map[string]Workload

type MetricManager struct {
	workloads map[string]Workloads
	lock      sync.RWMutex
	cache     cache.Cache
	stopCh    chan struct{}
}

func New(c cache.Cache) (*MetricManager, error) {
	ctrl := controller.New("metricCache", c, scheme.Scheme)
	ctrl.Watch(&appsv1.Deployment{})
	ctrl.Watch(&appsv1.DaemonSet{})
	ctrl.Watch(&appsv1.StatefulSet{})
	ctrl.Watch(&corev1.Namespace{})
	ctrl.Watch(&corev1.Pod{})
	stopCh := make(chan struct{})
	m := &MetricManager{
		workloads: make(map[string]Workloads),
		stopCh:    stopCh,
		cache:     c,
	}

	if err := m.initMetricManager(); err != nil {
		return nil, err
	}

	go ctrl.Start(stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	return m, nil
}

func (m *MetricManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, Metric{}, m)
}

func (m *MetricManager) initMetricManager() error {
	nses := &corev1.NamespaceList{}
	if err := m.cache.List(context.TODO(), nil, nses); err != nil {
		return fmt.Errorf("list namespace failed: %s\n", err.Error())
	}

	for _, ns := range nses.Items {
		if err := m.initPods(ns.Name); err != nil {
			return fmt.Errorf("list pods with namespace %s failed: %s", ns, err.Error())
		}
	}

	return nil
}

func (m *MetricManager) initPods(namespace string) error {
	pods := corev1.PodList{}
	if err := m.cache.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &pods); err != nil {
		if apierrors.IsNotFound(err) == false {
			return fmt.Errorf("list pods with namespace %s failed: %s", namespace, err.Error())
		}

		return nil
	}

	for _, pod := range pods.Items {
		if err := m.onCreatePod(&pod); err != nil {
			return err
		}
	}

	return nil
}

func (m *MetricManager) onCreatePod(pod *corev1.Pod) error {
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return nil
	}

	port, path, err := getWorkloadExposedMetric(pod.Annotations)
	if err != nil {
		return nil
	}

	workloads, ok := m.workloads[pod.Namespace]
	if ok == false {
		workloads = make(map[string]Workload)
		m.workloads[pod.Namespace] = workloads
	}

	ownerType, ownerName, err := helper.GetPodOwner(m.cache, pod)
	if err != nil {
		return fmt.Errorf("get pod %s owner with namespace %s failed: %s", pod.Name, pod.Namespace, err.Error())
	}

	workloadID := genWorkloadID(ownerType, ownerName)
	workload, ok := workloads[workloadID]
	if ok == false {
		workload = Workload{
			Type:       ownerType,
			Name:       ownerName,
			MetricPort: port,
			MetricPath: path,
		}
	} else {
		for _, p := range workload.Pods {
			if p.Name == pod.Name {
				return nil
			}
		}
	}

	workload.Pods = append(workload.Pods, Pod{
		Name: pod.Name,
		IP:   pod.Status.PodIP,
	})
	workloads[workloadID] = workload
	return nil
}

func genWorkloadID(typ, name string) string {
	return typ + "/" + name
}

func getWorkloadExposedMetric(annotations map[string]string) (int, string, error) {
	if scrape, ok := annotations[AnnotationsPrometheusScrape]; ok == false || scrape != "true" {
		return 0, "", fmt.Errorf("no set annotations %s", AnnotationsPrometheusScrape)
	}

	portStr, ok := annotations[AnnotationsPrometheusPort]
	if ok == false || portStr == "" {
		return 0, "", fmt.Errorf("no set annotations %s", AnnotationsPrometheusPort)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, "", fmt.Errorf("parse %s %s to integer failed: %s", AnnotationsPrometheusPort, portStr, err.Error())
	}

	path := annotations[AnnotationsPrometheusPath]
	if path == "" {
		path = DefaultMetricPath
	}

	return port, path, nil
}

func (m *MetricManager) List(ctx *resource.Context) interface{} {
	metrics, err := m.getMetrics(ctx)
	if err != nil {
		log.Warnf("list metrics failed:%s", err.Error())
		return nil
	}

	return metrics
}

func (m *MetricManager) getMetrics(ctx *resource.Context) (Metrics, error) {
	namespace := ctx.Resource.GetParent().GetParent().GetID()
	ownerType := ctx.Resource.GetParent().GetType()
	ownerName := ctx.Resource.GetParent().GetID()

	m.lock.RLock()
	workloads, ok := m.workloads[namespace]
	m.lock.RUnlock()
	if ok == false {
		return nil, fmt.Errorf("no found metrics with namespace %s", namespace)
	}

	if w, ok := workloads[genWorkloadID(ownerType, ownerName)]; ok {
		for _, pod := range w.Pods {
			if metrics, err := getPodMetrics(pod.IP, w.MetricPath, w.MetricPort); err != nil {
				log.Warnf("get pod %s metrics failed: %s", pod.Name, err.Error())
				continue
			} else {
				sort.Sort(metrics)
				return metrics, nil
			}
		}
	}

	return nil, fmt.Errorf("no found workload %s/%s metrics", ownerType, ownerName)
}

func getPodMetrics(podIP, metricPath string, metricPort int) (Metrics, error) {
	if podIP == "" {
		return nil, fmt.Errorf("pod ip is empty")
	}

	resp, err := http.Get(fmt.Sprintf(MetricsURL, podIP, metricPort, metricPath))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse metric family failed: %s", err.Error())
	}

	var metrics Metrics
	for _, mf := range metricFamilies {
		var metricFamilies []MetricFamily
		for _, m := range mf.GetMetric() {
			labels := make(map[string]string)
			for _, label := range m.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			if m.GetGauge() != nil || m.GetCounter() != nil {
				metricFamilies = append(metricFamilies, MetricFamily{
					Labels:  labels,
					Gauge:   Gauge{Value: int(m.GetGauge().GetValue())},
					Counter: Counter{Value: int(m.GetCounter().GetValue())},
				})
			}
		}

		if len(metricFamilies) != 0 {
			metrics = append(metrics, &Metric{
				Name:    mf.GetName(),
				Help:    mf.GetHelp(),
				Type:    mf.GetType().String(),
				Metrics: metricFamilies,
			})
		}
	}

	sort.Sort(metrics)
	return metrics, nil
}

func (m *MetricManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.Pod:
		m.onCreatePod(obj)
	}

	return handler.Result{}, nil
}

func (m *MetricManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		delete(m.workloads, obj.Name)
	case *appsv1.Deployment:
		m.onDeleteWorkload(obj.Spec.Template.Annotations, obj.Namespace, common.ResourceTypeDeployment, obj.Name)
	case *appsv1.DaemonSet:
		m.onDeleteWorkload(obj.Spec.Template.Annotations, obj.Namespace, common.ResourceTypeDaemonSet, obj.Name)
	case *appsv1.StatefulSet:
		m.onDeleteWorkload(obj.Spec.Template.Annotations, obj.Namespace, common.ResourceTypeStatefulSet, obj.Name)
	case *corev1.Pod:
		if _, _, err := getWorkloadExposedMetric(obj.Annotations); err == nil {
			m.onDeletePod(obj)
		}
	}

	return handler.Result{}, nil
}

func (m *MetricManager) onDeleteWorkload(annotions map[string]string, namespace, typ, name string) {
	if _, _, err := getWorkloadExposedMetric(annotions); err != nil {
		return
	}

	if workloads, ok := m.workloads[namespace]; ok {
		delete(workloads, genWorkloadID(typ, name))
	}
}

func (m *MetricManager) onDeletePod(pod *corev1.Pod) {
	workload, workloadID, ok := m.getWorkload(pod)
	if ok == false {
		return
	}

	for i, p := range workload.Pods {
		if p.Name == pod.Name {
			workload.Pods = append(workload.Pods[:i], workload.Pods[i+1:]...)
			break
		}
	}

	m.workloads[pod.Namespace][workloadID] = workload
}

func (m *MetricManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.ObjectNew.(type) {
	case *corev1.Pod:
		m.onUpdatePod(e.ObjectOld.(*corev1.Pod), obj)
	}
	return handler.Result{}, nil
}

func (m *MetricManager) onUpdatePod(oldPod *corev1.Pod, newPod *corev1.Pod) {
	if _, _, err := getWorkloadExposedMetric(newPod.Annotations); err != nil {
		return
	}

	if newPod.Status.Phase == corev1.PodSucceeded || newPod.Status.Phase == corev1.PodFailed {
		m.onDeletePod(newPod)
		return
	}

	if oldPod.Status.PodIP == newPod.Status.PodIP {
		return
	}

	workload, workloadID, ok := m.getWorkload(newPod)
	if ok == false {
		return
	}

	for i, pod := range workload.Pods {
		if pod.Name == newPod.Name {
			workload.Pods[i].IP = newPod.Status.PodIP
			break
		}
	}

	m.workloads[newPod.Namespace][workloadID] = workload
}

func (m *MetricManager) getWorkload(pod *corev1.Pod) (Workload, string, bool) {
	var workload Workload
	workloads, ok := m.workloads[pod.Namespace]
	if ok == false {
		return workload, "", false
	}

	ownerType, ownerName, err := helper.GetPodOwner(m.cache, pod)
	if err != nil {
		log.Warnf("get pod %s owner failed: %s", pod.Name, err.Error())
		return workload, "", false
	}

	workloadID := genWorkloadID(ownerType, ownerName)
	workload, ok = workloads[workloadID]
	if ok == false {
		return workload, "", false
	}

	return workload, workloadID, true
}

func (m *MetricManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

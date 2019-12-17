package metric

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/common/expfmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
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
)

const (
	MetricsURL                  = "http://%s:%d/%s"
	AnnotationsPrometheusPath   = "prometheus.io/path"
	AnnotationsPrometheusPort   = "prometheus.io/port"
	AnnotationsPrometheusScrape = "prometheus.io/scrape"
)

type Workloads map[string]map[string]Workload

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
		if err := m.initDeployments(ns.Name); err != nil {
			return fmt.Errorf("list deploy metrics with namespace %s failed: %s", ns, err.Error())
		}
		if err := m.initDaemonSets(ns.Name); err != nil {
			return fmt.Errorf("list daemonset metrics with namespace %s failed: %s", ns, err.Error())
		}
		if err := m.initStateFulSets(ns.Name); err != nil {
			return fmt.Errorf("list statefulset metrics with namespace %s failed: %s", ns, err.Error())
		}
	}

	return nil
}

func (m *MetricManager) initDeployments(namespace string) error {
	deploys := appsv1.DeploymentList{}
	if err := m.cache.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &deploys); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("list deploymets failed: %s", err.Error())
	}

	for _, deploy := range deploys.Items {
		m.onCreateWorkload(deploy.Spec.Template, namespace, deploy.Kind, deploy.Name, deploy.Spec.Selector)
	}

	return nil
}

func (m *MetricManager) onCreateWorkload(spec corev1.PodTemplateSpec, namespace, typ, name string, selector *metav1.LabelSelector) {
	port, path, err := getWorkloadExposedMetric(spec)
	if err != nil {
		log.Debugf("get workload exposed metric failed: %s", err.Error())
		return
	}

	if selector == nil {
		return
	}

	labelSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		log.Warnf("workload selector to label selector failed: %s", err.Error())
		return
	}

	workloads, ok := m.workloads[namespace]
	if ok == false {
		workloads = map[string]map[string]Workload{
			"deployment":  make(map[string]Workload),
			"daemonset":   make(map[string]Workload),
			"statefulset": make(map[string]Workload),
		}
		m.workloads[namespace] = workloads
	}

	if w, ok := workloads[typ]; ok {
		if _, ok := w[name]; ok {
			return
		}
	}

	podList := corev1.PodList{}
	if err := m.cache.List(context.TODO(), &client.ListOptions{Namespace: namespace,
		LabelSelector: labelSelector}, &podList); err != nil {
		log.Warnf("list pods with namespace %s failed: %s", namespace, err.Error())
		return
	}

	var pods []Pod
	for _, pod := range podList.Items {
		pods = append(pods, Pod{
			Name: pod.Name,
			IP:   pod.Status.PodIP,
		})
	}

	workloads[typ] = map[string]Workload{
		name: Workload{
			Type:       typ,
			Name:       name,
			MetricPort: port,
			MetricPath: path,
			Pods:       pods,
		}}
}

func getWorkloadExposedMetric(spec corev1.PodTemplateSpec) (int, string, error) {
	if scrape, ok := spec.Annotations[AnnotationsPrometheusScrape]; ok == false || scrape != "true" {
		return 0, "", fmt.Errorf("no found annotions %s", AnnotationsPrometheusScrape)
	}

	path, ok := spec.Annotations[AnnotationsPrometheusPath]
	if ok == false || path == "" {
		return 0, "", fmt.Errorf("no found annotions %s", AnnotationsPrometheusPath)
	}

	portStr, ok := spec.Annotations[AnnotationsPrometheusScrape]
	if ok == false || portStr == "" {
		return 0, "", fmt.Errorf("no found annotions %s", AnnotationsPrometheusPort)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, "", fmt.Errorf("%s %s is invalid integer: %s", AnnotationsPrometheusPort, portStr, err.Error())
	}

	return port, path, nil
}

func (m *MetricManager) initDaemonSets(namespace string) error {
	daemonsets := appsv1.DaemonSetList{}
	if err := m.cache.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &daemonsets); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("list daemonsets failed: %s", err.Error())
	}

	for _, ds := range daemonsets.Items {
		m.onCreateWorkload(ds.Spec.Template, namespace, ds.Kind, ds.Name, ds.Spec.Selector)
	}

	return nil
}

func (m *MetricManager) initStateFulSets(namespace string) error {
	statefulsets := appsv1.StatefulSetList{}
	if err := m.cache.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &statefulsets); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("list statefulsets failed: %s", err.Error())
	}

	for _, sts := range statefulsets.Items {
		m.onCreateWorkload(sts.Spec.Template, namespace, sts.Kind, sts.Name, sts.Spec.Selector)
	}

	return nil
}

func (m *MetricManager) getPodIP(namespace, podName string) (string, error) {
	var pod corev1.Pod
	if err := m.cache.Get(context.TODO(), k8stypes.NamespacedName{namespace, podName}, &pod); err != nil {
		return "", err
	}

	return pod.Status.PodIP, nil
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

	if nw, ok := workloads[ownerType]; ok {
		if w, ok := nw[ownerName]; ok {
			for _, pod := range w.Pods {
				if metrics, err := getPodMetrics(pod.IP, w.MetricPath, w.MetricPort); err != nil {
					log.Warnf("get pod %s metric name failed: %s", pod.Name, err.Error())
					continue
				} else {
					sort.Sort(metrics)
					return metrics, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no found workload %s/%s metrics", ownerType, ownerName)
}

func getPodMetrics(podIP, metricPath string, metricPort int) (Metrics, error) {
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
		found := false
		for _, m := range mf.GetMetric() {
			if m.GetGauge() != nil || m.GetCounter() != nil {
				found = true
				break
			}
		}

		if found {
			metrics = append(metrics, &Metric{
				Name: mf.GetName(),
			})
		}
	}

	sort.Sort(metrics)
	return metrics, nil
}

func (m *MetricManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	switch obj := e.Object.(type) {
	case *appsv1.Deployment:
		m.onCreateWorkload(obj.Spec.Template, obj.Namespace, obj.Kind, obj.Name, obj.Spec.Selector)
	case *appsv1.DaemonSet:
		m.onCreateWorkload(obj.Spec.Template, obj.Namespace, obj.Kind, obj.Name, obj.Spec.Selector)
	case *appsv1.StatefulSet:
		m.onCreateWorkload(obj.Spec.Template, obj.Namespace, obj.Kind, obj.Name, obj.Spec.Selector)
	}

	return handler.Result{}, nil
}

func (m *MetricManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		delete(m.workloads, obj.Name)
	case *appsv1.Deployment:
		m.onDeleteWorkload(obj.Spec.Template, obj.Namespace, obj.Kind, obj.Name)
	case *appsv1.DaemonSet:
		m.onDeleteWorkload(obj.Spec.Template, obj.Namespace, obj.Kind, obj.Name)
	case *appsv1.StatefulSet:
		m.onDeleteWorkload(obj.Spec.Template, obj.Namespace, obj.Kind, obj.Name)
	}

	return handler.Result{}, nil
}

func (m *MetricManager) onDeleteWorkload(spec corev1.PodTemplateSpec, namespace, typ, name string) {
	if _, _, err := getWorkloadExposedMetric(spec); err != nil {
		return
	}

	workloads, ok := m.workloads[namespace]
	if ok == false {
		return
	}

	delete(workloads[typ], name)
}

func (m *MetricManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch obj := e.ObjectNew.(type) {
	case *corev1.Pod:
		m.onUpdatePod(e.ObjectOld.(*corev1.Pod), obj)
	}
	return handler.Result{}, nil
}

func (m *MetricManager) onUpdatePod(oldPod *corev1.Pod, newPod *corev1.Pod) {
	if oldPod.Status.PodIP == newPod.Status.PodIP {
		return
	}

	m.lock.RLock()
	workloads, ok := m.workloads[newPod.Namespace]
	m.lock.RUnlock()
	if ok == false {
		return
	}

	ownerType, ownerName, err := helper.GetPodOwner(m.cache, newPod)
	if err != nil {
		log.Warnf("get pod %s owner failed: %s", newPod.Name, err.Error())
		return
	}

	if nw, ok := workloads[strings.ToLower(ownerType)]; ok {
		if w, ok := nw[ownerName]; ok {
			for i, pod := range w.Pods {
				if pod.Name == newPod.Name {
					w.Pods[i].IP = newPod.Status.PodIP
					break
				}
			}
		}
	}
}

func (m *MetricManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

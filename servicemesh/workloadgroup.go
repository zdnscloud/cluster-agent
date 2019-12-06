package servicemesh

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/yourbasic/graph"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/helper"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

const (
	LinkerdControllerSVCAPIAddr = "http://linkerd-controller-api.linkerd.svc:8085/api/v1/"
	LinkerdControllerAPIAddr    = "http://10.42.1.160:8085/api/v1/"
	Deployment                  = "deployment"
	DaemonSet                   = "daemonset"
	StatefulSet                 = "statefulset"
	DeploymentPrefix            = "dpm-"
	DaemonSetPrefix             = "dms-"
	StatefulSetPrefix           = "sts-"
	ProxyContainerName          = "linkerd-proxy"
	ProxyContainerPortName      = "linkerd-admin"
	LinkerdInjectAnnotation     = "linkerd.io/inject"
)

type WorkloadGroupOptions struct {
	Group     []string
	Namespace string
}

type InjectedResouces struct {
	podOwners map[string]string
	workloads map[string][]string
}

type WorkloadGroupManager struct {
	nsResources  map[string]InjectedResouces
	apiServerURL *url.URL
	lock         sync.RWMutex
	cache        cache.Cache
	stopCh       chan struct{}
}

func New(c cache.Cache) (*WorkloadGroupManager, error) {
	ctrl := controller.New("workloadgroupsCache", c, scheme.Scheme)
	ctrl.Watch(&appsv1.Deployment{})
	ctrl.Watch(&appsv1.DaemonSet{})
	ctrl.Watch(&appsv1.StatefulSet{})
	ctrl.Watch(&corev1.Namespace{})
	ctrl.Watch(&corev1.Pod{})
	stopCh := make(chan struct{})
	apiServerURL, err := url.Parse(LinkerdControllerAPIAddr)
	if err != nil {
		return nil, fmt.Errorf("new linkerd public api server url failed: %s", err.Error())
	}

	m := &WorkloadGroupManager{
		nsResources:  make(map[string]InjectedResouces),
		apiServerURL: apiServerURL,
		stopCh:       stopCh,
		cache:        c,
	}

	if err := m.initWorkloadGroupManager(); err != nil {
		return nil, err
	}

	go ctrl.Start(stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	return m, nil
}

func (m *WorkloadGroupManager) initWorkloadGroupManager() error {
	nses := &corev1.NamespaceList{}
	if err := m.cache.List(context.TODO(), nil, nses); err != nil {
		return fmt.Errorf("list namespace failed: %s\n", err.Error())
	}

	for _, ns := range nses.Items {
		pods := &corev1.PodList{}
		if err := m.cache.List(context.TODO(), &client.ListOptions{Namespace: ns.Name}, pods); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("list pods with namespace %s failed: %s", ns, err.Error())
		}

		for _, pod := range pods.Items {
			if err := m.onCreatePod(&pod); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *WorkloadGroupManager) onCreatePod(pod *corev1.Pod) error {
	if isLinkerdInjectedPod(pod) == false {
		return nil
	}

	ownerType, ownerName, err := helper.GetPodOwner(m.cache, pod)
	if err != nil {
		return fmt.Errorf("get pod %s owner with namespace %s failed: %s", pod.Name, pod.Namespace, err.Error())
	}

	workloadId, ok := genWorkloadID(ownerType, ownerName)
	if ok == false {
		return nil
	}

	resources, ok := m.nsResources[pod.Namespace]
	if ok == false {
		resources = InjectedResouces{
			podOwners: make(map[string]string),
			workloads: make(map[string][]string),
		}
		m.nsResources[pod.Namespace] = resources
	}
	resources.podOwners[pod.Name] = workloadId
	resources.workloads[workloadId] = append(resources.workloads[workloadId], pod.Name)
	return nil
}

func isLinkerdInjectedPod(pod *corev1.Pod) bool {
	if enabled, ok := pod.Annotations[LinkerdInjectAnnotation]; ok && enabled == "enabled" {
		return true
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == ProxyContainerName {
			for _, p := range c.Ports {
				if p.Name == ProxyContainerPortName {
					return true
				}
			}
		}
	}

	return false
}

func genWorkloadID(typ, name string) (string, bool) {
	var workloadPrefix string
	switch strings.ToLower(typ) {
	case Deployment:
		workloadPrefix = DeploymentPrefix
	case DaemonSet:
		workloadPrefix = DaemonSetPrefix
	case StatefulSet:
		workloadPrefix = StatefulSetPrefix
	default:
		return "", false
	}

	return workloadPrefix + name, true
}

func (m *WorkloadGroupManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, types.WorkloadGroup{}, m)
	schemas.MustImport(version, types.Workload{}, newWorkloadManager(m.apiServerURL, m))
	schemas.MustImport(version, types.Pod{}, newPodManager(m.apiServerURL, m))
}

func (m *WorkloadGroupManager) List(ctx *resource.Context) interface{} {
	namespace := ctx.Resource.GetParent().GetID()
	groups, err := m.getWorkloadGroups(namespace)
	if err != nil {
		log.Warnf("list workload groups failed: %s", err.Error())
		return nil
	}

	return groups
}

func (m *WorkloadGroupManager) getWorkloadGroups(namespace string) (types.WorkloadGroups, error) {
	optionsGroups, err := m.getStatOptionsGroups(namespace)
	if err != nil {
		return nil, err
	}

	resultCh, err := errgroup.Batch(optionsGroups, func(options interface{}) (interface{}, error) {
		return m.getWorkloadGroup(options.([]*StatOptions))
	})
	if err != nil {
		return nil, err
	}

	var workloadgroups types.WorkloadGroups
	for result := range resultCh {
		workloadgroups = append(workloadgroups, result.(*types.WorkloadGroup))
	}
	sort.Sort(workloadgroups)
	return workloadgroups, nil
}

func (m *WorkloadGroupManager) getWorkloadGroup(statOptions []*StatOptions) (*types.WorkloadGroup, error) {
	resultCh, err := errgroup.Batch(statOptions, func(options interface{}) (interface{}, error) {
		opts := options.(*StatOptions)
		stat, err := getStat(opts)
		if err != nil {
			return nil, err
		}

		workload := &types.Workload{Stat: stat}
		workload.SetID(opts.ResourceID)
		return workload, nil
	})
	if err != nil {
		return nil, err
	}

	workloadgroup := &types.WorkloadGroup{}
	for result := range resultCh {
		workloadgroup.Workloads = append(workloadgroup.Workloads, result.(*types.Workload))
	}

	id, err := uuid.Gen()
	if err != nil {
		return nil, fmt.Errorf("gen workload group id failed: %s", err.Error())
	}

	workloadgroup.SetID(id)
	sort.Sort(workloadgroup.Workloads)
	return workloadgroup, nil
}

func (m *WorkloadGroupManager) getStatOptionsGroups(namespace string) ([][]*StatOptions, error) {
	m.lock.RLock()
	resources, ok := m.nsResources[namespace]
	m.lock.RUnlock()
	if ok == false {
		return nil, nil
	}

	es, err := getEdges(m.apiServerURL, namespace, Pod)
	if err != nil {
		return nil, err
	}

	id := 0
	workloadIDs := make(map[string]int)
	var edges types.Edges
	for _, e := range es {
		src, ok := resources.podOwners[e.Src.Name]
		if ok == false {
			continue
		}

		dst, ok := resources.podOwners[e.Dst.Name]
		if ok == false {
			continue
		}

		if _, ok := workloadIDs[src]; ok == false {
			workloadIDs[src] = id
			id += 1
		}

		if _, ok := workloadIDs[dst]; ok == false {
			workloadIDs[dst] = id
			id += 1
		}

		edges = append(edges, &types.Edge{
			Src: types.Resource{
				Name: src,
			},
			Dst: types.Resource{
				Name: dst,
			},
		})
	}

	g := graph.New(id)
	for _, e := range edges {
		g.Add(workloadIDs[e.Src.Name], workloadIDs[e.Dst.Name])
	}

	var optionsGroups [][]*StatOptions
	for _, ids := range graph.Components(g) {
		var options []*StatOptions
		for _, id := range ids {
			for wId, _id := range workloadIDs {
				if _id == id {
					options = append(options, m.workloadIDToStatOptions(namespace, wId))
					break
				}
			}
		}

		optionsGroups = append(optionsGroups, options)
	}

	for wId := range resources.workloads {
		if _, ok := workloadIDs[wId]; ok == false {
			optionsGroups = append(optionsGroups, []*StatOptions{m.workloadIDToStatOptions(namespace, wId)})
		}
	}

	return optionsGroups, nil
}

func (m *WorkloadGroupManager) workloadIDToStatOptions(namespace, id string) *StatOptions {
	resourceType, resourceName, _ := getResourceTypeAndName(id)
	return &StatOptions{
		ApiServerURL: m.apiServerURL,
		Namespace:    namespace,
		ResourceID:   id,
		ResourceType: resourceType,
		ResourceName: resourceName,
	}
}

func (m *WorkloadGroupManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.Pod:
		m.onCreatePod(obj)
	}

	return handler.Result{}, nil
}

func (m *WorkloadGroupManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		delete(m.nsResources, obj.Name)
	case *appsv1.Deployment:
		delete(m.nsResources[obj.Namespace].workloads, DeploymentPrefix+obj.Name)
	case *appsv1.DaemonSet:
		delete(m.nsResources[obj.Namespace].workloads, DaemonSetPrefix+obj.Name)
	case *appsv1.StatefulSet:
		delete(m.nsResources[obj.Namespace].workloads, StatefulSetPrefix+obj.Name)
	case *corev1.Pod:
		delete(m.nsResources[obj.Namespace].podOwners, obj.Name)
	}

	return handler.Result{}, nil
}

func (m *WorkloadGroupManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (m *WorkloadGroupManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (m *WorkloadGroupManager) GetWorkloadPods(namespace, workloadId string) ([]string, error) {
	m.lock.RLock()
	resources, ok := m.nsResources[namespace]
	m.lock.RUnlock()
	if ok == false {
		return nil, fmt.Errorf("not found namespace %s", namespace)
	}

	if pods, ok := resources.workloads[workloadId]; ok == false {
		return nil, fmt.Errorf("not found workload id %s with namespace %s", workloadId, namespace)
	} else {
		return pods, nil
	}
}

func (m *WorkloadGroupManager) IsPodBelongToWorkload(namespace, workloadId, podName string) error {
	m.lock.RLock()
	resources, ok := m.nsResources[namespace]
	m.lock.RUnlock()
	if ok == false {
		return fmt.Errorf("not found namespace %s", namespace)
	}

	if _, ok := resources.workloads[workloadId]; ok == false {
		return fmt.Errorf("not found workload id %s with namespace %s", workloadId, namespace)
	}

	if wid, ok := resources.podOwners[podName]; ok == false {
		return fmt.Errorf("not found pod %s with namespace %s", podName, namespace)
	} else if wid != workloadId {
		return fmt.Errorf("pod %s with namespace %s belong to workload id %s not %s", podName, namespace, wid, workloadId)
	}

	return nil
}

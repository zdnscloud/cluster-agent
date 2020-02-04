package servicemesh

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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
	"github.com/zdnscloud/cement/set"
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
	LinkerdControllerAPIAddr = "http://linkerd-controller-api.zcloud.svc:8085/api/v1/"
	ResourceTypeDeployment   = "deployment"
	ResourceTypeDaemonSet    = "daemonset"
	ResourceTypeStatefulSet  = "statefulset"
	DeploymentPrefix         = "dpm-"
	DaemonSetPrefix          = "dms-"
	StatefulSetPrefix        = "sts-"
	ProxyContainerName       = "linkerd-proxy"
	ProxyContainerPortName   = "linkerd-admin"
	LinkerdInjectAnnotation  = "linkerd.io/inject"
)

type InjectedResouces struct {
	podOwners    map[string]string
	workloadPods map[string][]string
}

type WorkloadManager struct {
	nsResources  map[string]InjectedResouces
	apiServerURL *url.URL
	lock         sync.RWMutex
	cache        cache.Cache
	stopCh       chan struct{}
}

func (m *WorkloadManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, types.SvcMeshWorkload{}, m)
	schemas.MustImport(version, types.SvcMeshPod{}, newPodManager(m.apiServerURL, m))
}

func New(c cache.Cache) (*WorkloadManager, error) {
	ctrl := controller.New("svcmeshWorkloadCache", c, scheme.Scheme)
	ctrl.Watch(&appsv1.Deployment{})
	ctrl.Watch(&appsv1.DaemonSet{})
	ctrl.Watch(&appsv1.StatefulSet{})
	ctrl.Watch(&corev1.Namespace{})
	ctrl.Watch(&corev1.Pod{})
	stopCh := make(chan struct{})
	apiServerURL, err := url.Parse(LinkerdControllerAPIAddr)
	if err != nil {
		return nil, fmt.Errorf("new servicemesh public api server url failed: %s", err.Error())
	}

	m := &WorkloadManager{
		nsResources:  make(map[string]InjectedResouces),
		apiServerURL: apiServerURL,
		stopCh:       stopCh,
		cache:        c,
	}

	if err := m.initWorkloadManager(); err != nil {
		return nil, err
	}

	go ctrl.Start(stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	return m, nil
}

func (m *WorkloadManager) initWorkloadManager() error {
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

func (m *WorkloadManager) onCreatePod(pod *corev1.Pod) error {
	if (pod.Status.Phase != corev1.PodPending && pod.Status.Phase != corev1.PodRunning) ||
		isLinkerdInjectedPod(pod) == false {
		return nil
	}

	resources, ok := m.nsResources[pod.Namespace]
	if ok == false {
		resources = InjectedResouces{
			podOwners:    make(map[string]string),
			workloadPods: make(map[string][]string),
		}
		m.nsResources[pod.Namespace] = resources
	} else if _, ok := resources.podOwners[pod.Name]; ok {
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

	resources.podOwners[pod.Name] = workloadId
	resources.workloadPods[workloadId] = append(resources.workloadPods[workloadId], pod.Name)
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
	case ResourceTypeDeployment:
		workloadPrefix = DeploymentPrefix
	case ResourceTypeDaemonSet:
		workloadPrefix = DaemonSetPrefix
	case ResourceTypeStatefulSet:
		workloadPrefix = StatefulSetPrefix
	default:
		return "", false
	}

	return workloadPrefix + name, true
}

func (m *WorkloadManager) List(ctx *resource.Context) interface{} {
	namespace := ctx.Resource.GetParent().GetID()
	workloads, err := m.getWorkloads(namespace)
	if err != nil {
		log.Warnf("list workloads failed: %s", err.Error())
		return nil
	}

	return workloads
}

func (m *WorkloadManager) getWorkloads(namespace string) (types.SvcMeshWorkloads, error) {
	optionsGroups, err := m.getStatOptionsGroups(namespace)
	if err != nil {
		return nil, err
	}

	resultCh, err := errgroup.Batch(optionsGroups, func(options interface{}) (interface{}, error) {
		return m.getWorkloadsGroup(options.([]*StatOption))
	})
	if err != nil {
		return nil, err
	}

	var groups []*WorkloadGroup
	for result := range resultCh {
		groups = append(groups, result.(*WorkloadGroup))
	}

	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Workloads) == len(groups[j].Workloads) {
			return groups[i].Workloads[0].Stat.ID < groups[j].Workloads[0].Stat.ID
		}

		return len(groups[j].Workloads) < len(groups[i].Workloads)
	})
	var workloads types.SvcMeshWorkloads
	for _, group := range groups {
		workloads = append(workloads, group.Workloads...)
	}
	return workloads, nil
}

func (m *WorkloadManager) getStatOptionsGroups(namespace string) ([][]*StatOption, error) {
	m.lock.RLock()
	resources, ok := m.nsResources[namespace]
	m.lock.RUnlock()
	if ok == false {
		return nil, fmt.Errorf("namespace %s no resources injected by servicemesh", namespace)
	}

	es, err := getEdges(m.apiServerURL, namespace, ResourceTypePod)
	if err != nil {
		return nil, err
	}

	id := 0
	workloadIDs := make(map[string]int)
	idWorkloads := make(map[int]string)
	srcDsts := make(map[string]set.StringSet)
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
			idWorkloads[id] = src
			id += 1
		}

		if _, ok := workloadIDs[dst]; ok == false {
			workloadIDs[dst] = id
			idWorkloads[id] = dst
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

		ss, ok := srcDsts[src]
		if ok == false {
			ss = set.NewStringSet()
			srcDsts[src] = ss
		}

		ss.Add(dst)
	}

	g := graph.New(id)
	for _, e := range edges {
		g.Add(workloadIDs[e.Src.Name], workloadIDs[e.Dst.Name])
	}

	var optionsGroups [][]*StatOption
	for _, ids := range graph.Components(g) {
		var options []*StatOption
		for _, id := range ids {
			wId := idWorkloads[id]
			options = append(options, m.genGroupStatOption(namespace, wId, srcDsts[wId].ToSortedSlice()))
		}

		optionsGroups = append(optionsGroups, options)
	}

	for wId := range resources.workloadPods {
		if _, ok := workloadIDs[wId]; ok == false {
			optionsGroups = append(optionsGroups, []*StatOption{m.genGroupStatOption(namespace, wId, nil)})
		}
	}

	return optionsGroups, nil
}

func (m *WorkloadManager) genGroupStatOption(namespace, id string, dsts []string) *StatOption {
	resourceType, resourceName, _ := getResourceTypeAndName(id)
	return genStatOption(m.apiServerURL, namespace, resourceType, resourceName, dsts, false, false)
}

func genStatOption(apiUrl *url.URL, namespace, resourceType, resourceName string, dsts []string, from, to bool) *StatOption {
	return &StatOption{
		ApiServerURL: apiUrl,
		Namespace:    namespace,
		Dsts:         dsts,
		ResourceType: resourceType,
		ResourceName: resourceName,
		From:         from,
		To:           to,
	}
}

type WorkloadGroup struct {
	Workloads types.SvcMeshWorkloads
}

func (m *WorkloadManager) getWorkloadsGroup(statOptions []*StatOption) (*WorkloadGroup, error) {
	resultCh, err := errgroup.Batch(statOptions, func(option interface{}) (interface{}, error) {
		return getWorkloadWithOption(option.(*StatOption))
	})
	if err != nil {
		return nil, err
	}

	var workloads types.SvcMeshWorkloads
	var workloadIDs []string
	for result := range resultCh {
		if r := result.(*StatResult); r.RequestType == RequestTypeWorkload {
			workload := &types.SvcMeshWorkload{Destinations: r.Destinations, Stat: r.Stat}
			workload.SetID(workload.Stat.ID)
			workloads = append(workloads, workload)
			workloadIDs = append(workloadIDs, workload.Stat.ID)
		}
	}

	groupID, err := genWorkloadGroupID(workloadIDs)
	if err != nil {
		return nil, fmt.Errorf("gen workload group id failed: %s", err.Error())
	}

	for _, workload := range workloads {
		workload.GroupID = groupID
	}

	sort.Sort(workloads)
	return &WorkloadGroup{Workloads: workloads}, nil
}

func genWorkloadGroupID(ids []string) (string, error) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	data, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func (m *WorkloadManager) Get(ctx *resource.Context) resource.Resource {
	namespace := ctx.Resource.GetParent().GetID()
	id := ctx.Resource.(*types.SvcMeshWorkload).GetID()
	workload, err := m.getWorkload(namespace, id)
	if err != nil {
		log.Warnf("get workload with id %s failed: %s", id, err.Error())
		return nil
	}

	return workload
}

func (m *WorkloadManager) getWorkload(namespace, id string) (*types.SvcMeshWorkload, error) {
	statOptions, err := m.genStatOptions(namespace, id)
	if err != nil {
		return nil, err
	}

	resultCh, err := errgroup.Batch(statOptions, func(option interface{}) (interface{}, error) {
		return getWorkloadWithOption(option.(*StatOption))
	})
	if err != nil {
		return nil, err
	}

	workload := &types.SvcMeshWorkload{}
	for result := range resultCh {
		switch r := result.(*StatResult); r.RequestType {
		case RequestTypeInbound:
			workload.Inbound = r.Inbound
		case RequestTypeOutbound:
			workload.Outbound = r.Outbound
		case RequestTypeWorkload:
			workload.Stat = r.Stat
		case RequestTypePod:
			pod := &types.SvcMeshPod{Stat: r.Stat}
			pod.SetID(pod.Stat.Resource.Name)
			workload.Pods = append(workload.Pods, pod)
		}
	}

	sort.Sort(workload.Pods)
	workload.SetID(id)
	return workload, nil
}

func (m *WorkloadManager) genStatOptions(namespace, id string) ([]*StatOption, error) {
	resourceType, resourceName, err := getResourceTypeAndName(id)
	if err != nil {
		return nil, err
	}

	options := genBasicStatOptions(m.apiServerURL, namespace, resourceType, resourceName)
	pods, err := m.getWorkloadPods(namespace, id)
	if err != nil {
		return nil, err
	}

	for _, podName := range pods {
		options = append(options, genStatOption(m.apiServerURL, namespace, ResourceTypePod, podName, nil, false, false))
	}

	return options, nil
}

func getResourceTypeAndName(id string) (string, string, error) {
	if len(id) <= 4 {
		return "", "", fmt.Errorf("invalid workload id, len must be longer than 4")
	}

	prefix := id[:4]
	name := id[4:]
	var typ string
	switch prefix {
	case DeploymentPrefix:
		typ = ResourceTypeDeployment
	case DaemonSetPrefix:
		typ = ResourceTypeDaemonSet
	case StatefulSetPrefix:
		typ = ResourceTypeStatefulSet
	default:
		return "", "", fmt.Errorf("unspported workload prefix %s", prefix)
	}

	return typ, name, nil
}

func (m *WorkloadManager) getWorkloadPods(namespace, workloadId string) ([]string, error) {
	m.lock.RLock()
	resources, ok := m.nsResources[namespace]
	m.lock.RUnlock()
	if ok == false {
		return nil, fmt.Errorf("namespace %s no resources injected by servicemesh", namespace)
	}

	if pods, ok := resources.workloadPods[workloadId]; ok == false {
		return nil, fmt.Errorf("not found svcmeshworkload id %s with namespace %s", workloadId, namespace)
	} else {
		return pods, nil
	}
}

func genBasicStatOptions(apiServerURL *url.URL, namespace, resourceType, resourceName string) []*StatOption {
	return []*StatOption{
		genStatOption(apiServerURL, namespace, resourceType, resourceName, nil, false, false),
		genStatOption(apiServerURL, namespace, resourceType, resourceName, nil, true, false),
		genStatOption(apiServerURL, namespace, resourceType, resourceName, nil, false, true),
	}
}

type RequestType string

const (
	RequestTypeInbound  RequestType = "inbound"
	RequestTypeOutbound RequestType = "outbound"
	RequestTypeWorkload RequestType = "workload"
	RequestTypePod      RequestType = "pod"
)

type StatResult struct {
	RequestType  RequestType
	Destinations []string
	Stat         types.Stat
	Inbound      types.Stats
	Outbound     types.Stats
}

func getWorkloadWithOption(opt *StatOption) (*StatResult, error) {
	if opt.From {
		stats, err := getStats(opt)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s outbound stats with namespace %s failed: %s",
				opt.ResourceType, opt.ResourceName, opt.Namespace, err.Error())
		}

		return &StatResult{RequestType: RequestTypeOutbound, Outbound: stats}, nil
	} else if opt.To {
		stats, err := getStats(opt)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s inbound stats with namespace %s failed: %s",
				opt.ResourceType, opt.ResourceName, opt.Namespace, err.Error())
		}

		return &StatResult{RequestType: RequestTypeInbound, Inbound: stats}, nil
	} else {
		stat, err := getStat(opt)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s stats with namespace %s failed: %s",
				opt.ResourceType, opt.ResourceName, opt.Namespace, err.Error())
		}

		typ := RequestTypeWorkload
		if opt.ResourceType == ResourceTypePod {
			typ = RequestTypePod
		}

		return &StatResult{RequestType: typ, Destinations: opt.Dsts, Stat: stat}, nil
	}
}

func (m *WorkloadManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.Pod:
		m.onCreatePod(obj)
	}

	return handler.Result{}, nil
}

func (m *WorkloadManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		delete(m.nsResources, obj.Name)
	case *appsv1.Deployment:
		m.onDeleteWorkload(obj.Namespace, DeploymentPrefix+obj.Name)
	case *appsv1.DaemonSet:
		m.onDeleteWorkload(obj.Namespace, DaemonSetPrefix+obj.Name)
	case *appsv1.StatefulSet:
		m.onDeleteWorkload(obj.Namespace, StatefulSetPrefix+obj.Name)
	case *corev1.Pod:
		m.onDeletePod(obj)
	}

	return handler.Result{}, nil
}

func (m *WorkloadManager) onDeleteWorkload(namespace, workloadId string) {
	resources, ok := m.nsResources[namespace]
	if ok == false {
		return
	}

	pods, ok := resources.workloadPods[workloadId]
	if ok == false {
		return
	}

	delete(resources.workloadPods, workloadId)
	for _, pod := range pods {
		delete(resources.podOwners, pod)
	}
}

func (m *WorkloadManager) onDeletePod(pod *corev1.Pod) {
	if isLinkerdInjectedPod(pod) == false {
		return
	}

	resources, ok := m.nsResources[pod.Namespace]
	if ok == false {
		return
	}

	delete(resources.podOwners, pod.Name)
	ownerType, ownerName, err := helper.GetPodOwner(m.cache, pod)
	if err != nil {
		log.Warnf("get pod %s owner with namespace %s failed: %s", pod.Name, pod.Namespace, err.Error())
		return
	}

	workloadId, ok := genWorkloadID(ownerType, ownerName)
	if ok == false {
		return
	}

	for i, podName := range resources.workloadPods[workloadId] {
		if podName == pod.Name {
			resources.workloadPods[workloadId] = append(resources.workloadPods[workloadId][:i],
				resources.workloadPods[workloadId][i+1:]...)
			break
		}
	}
}

func (m *WorkloadManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch obj := e.ObjectNew.(type) {
	case *corev1.Pod:
		m.OnUpdatePod(obj)
	}
	return handler.Result{}, nil
}

func (m *WorkloadManager) OnUpdatePod(k8spod *corev1.Pod) {
	if k8spod.Status.Phase == corev1.PodSucceeded || k8spod.Status.Phase == corev1.PodFailed {
		m.lock.Lock()
		m.onDeletePod(k8spod)
		m.lock.Unlock()
	}
}

func (m *WorkloadManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (m *WorkloadManager) GetPodOwners(namespace, workloadId, podName string) (map[string]string, error) {
	m.lock.RLock()
	resources, ok := m.nsResources[namespace]
	m.lock.RUnlock()
	if ok == false {
		return nil, fmt.Errorf("namespace %s no resources injected by servicemesh", namespace)
	}

	if _, ok := resources.workloadPods[workloadId]; ok == false {
		return nil, fmt.Errorf("not found svcmeshworkload id %s with namespace %s", workloadId, namespace)
	}

	if wid, ok := resources.podOwners[podName]; ok == false {
		return nil, fmt.Errorf("not found pod %s with namespace %s", podName, namespace)
	} else if wid != workloadId {
		return nil, fmt.Errorf("pod %s with namespace %s belong to svcmeshworkload %s not %s", podName, namespace, wid, workloadId)
	}

	return resources.podOwners, nil
}

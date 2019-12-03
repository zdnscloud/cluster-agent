package servicemesh

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/yourbasic/graph"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

const (
	LinkerdControllerAPIAddr = "http://linkerd-controller-api.linkerd.svc:8085/api/v1/"
	Deployment               = "deployment"
	DaemonSet                = "daemonset"
	StatefulSet              = "statefulset"
	DeploymentPrefix         = "dpm"
	DaemonSetPrefix          = "dms"
	StatefulSetPrefix        = "sts"
	ProxyContainerName       = "linkerd-proxy"
	ProxyContainerPortName   = "linkerd-admin"
	LinkerdInjectAnnotation  = "linkerd.io/inject"
)

var WorkloadKinds = []string{Deployment, DaemonSet, StatefulSet}

type WorkloadGroupManager struct {
	apiServerURL *url.URL
	client       client.Client
}

func New(cli client.Client) (*WorkloadGroupManager, error) {
	apiServerURL, err := url.Parse(LinkerdControllerAPIAddr)
	if err != nil {
		return nil, fmt.Errorf("new linkerd public api server url failed: %s", err.Error())
	}

	return &WorkloadGroupManager{apiServerURL: apiServerURL, client: cli}, nil
}

func (m *WorkloadGroupManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, types.WorkloadGroup{}, m)
	schemas.MustImport(version, types.Workload{}, newWorkloadManager(m.apiServerURL, m.client))
	schemas.MustImport(version, types.Pod{}, newPodManager(m.apiServerURL))
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
	groups, err := m.getResourceGroups(namespace)
	if err != nil {
		return nil, err
	}

	var workloadgroups types.WorkloadGroups
	for _, group := range groups {
		workloadgroup := &types.WorkloadGroup{}
		for _, r := range group {
			stat, err := getStat(m.apiServerURL, namespace, r.Type, r.Name)
			if err != nil {
				return nil, err
			}

			workload := &types.Workload{Stat: stat}
			workload.SetID(genWorkloadID(r.Type, r.Name))
			workloadgroup.Workloads = append(workloadgroup.Workloads, workload)
		}

		if len(workloadgroup.Workloads) != 0 {
			id, err := uuid.Gen()
			if err != nil {
				return nil, fmt.Errorf("gen workload group id failed: %s", err.Error())
			}

			workloadgroup.SetID(id)
			sort.Sort(workloadgroup.Workloads)
			workloadgroups = append(workloadgroups, workloadgroup)
		}
	}

	sort.Sort(workloadgroups)
	return workloadgroups, nil
}

func genWorkloadID(typ, name string) string {
	workloadPrefix := ""
	switch typ {
	case Deployment:
		workloadPrefix = DeploymentPrefix
	case DaemonSet:
		workloadPrefix = DaemonSetPrefix
	case StatefulSet:
		workloadPrefix = StatefulSetPrefix
	}

	return workloadPrefix + name
}

func (m *WorkloadGroupManager) getResourceGroups(namespace string) ([][]types.Resource, error) {
	resources, err := m.getWorkloadResources(namespace)
	if err != nil {
		return nil, err
	}

	id := 0
	edges := make(types.Edges, 0)
	resourceIDs := make(map[string]int)
	for _, kind := range WorkloadKinds {
		es, err := getEdges(m.apiServerURL, namespace, kind)
		if err != nil {
			return nil, err
		}

		for _, e := range es {
			edges = append(edges, e)
			src := resourceToString(e.Src)
			if _, ok := resourceIDs[src]; ok == false {
				resourceIDs[src] = id
				id += 1
			}

			dst := resourceToString(e.Dst)
			if _, ok := resourceIDs[dst]; ok == false {
				resourceIDs[dst] = id
				id += 1
			}
		}
	}

	g := graph.New(id)
	for _, e := range edges {
		g.Add(resourceIDs[resourceToString(e.Src)], resourceIDs[resourceToString(e.Dst)])
	}

	var resourceGroups [][]types.Resource
	for _, ids := range graph.Components(g) {
		var rs []types.Resource
		for _, id := range ids {
			for str, wId := range resourceIDs {
				if wId == id {
					rs = append(rs, resourceFromString(str))
					break
				}
			}
		}

		resourceGroups = append(resourceGroups, rs)
	}

	for _, r := range resources {
		if _, ok := resourceIDs[resourceToString(r)]; ok == false {
			resourceGroups = append(resourceGroups, []types.Resource{r})
		}
	}

	return resourceGroups, nil
}

func resourceToString(r types.Resource) string {
	return r.Type + "/" + r.Name
}

func resourceFromString(str string) types.Resource {
	typeAndName := strings.SplitN(str, "/", 2)
	return types.Resource{
		Type: typeAndName[0],
		Name: typeAndName[1],
	}
}

func (m *WorkloadGroupManager) getWorkloadResources(namespace string) ([]types.Resource, error) {
	var resources []types.Resource

	deploys := appsv1.DeploymentList{}
	if err := m.client.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &deploys); err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, fmt.Errorf("list deployments with namespace %s failed: %s", namespace, err.Error())
		}
	}

	for _, deploy := range deploys.Items {
		if isMeshedWorkload(deploy.Spec.Template) {
			resources = append(resources, types.Resource{
				Type: Deployment,
				Name: deploy.Name,
			})
		}
	}

	dss := appsv1.DaemonSetList{}
	if err := m.client.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &dss); err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, fmt.Errorf("list daemonsets with namespace %s failed: %s", namespace, err.Error())
		}
	}

	for _, ds := range dss.Items {
		if isMeshedWorkload(ds.Spec.Template) {
			resources = append(resources, types.Resource{
				Type: DaemonSet,
				Name: ds.Name,
			})
		}
	}

	stss := appsv1.StatefulSetList{}
	if err := m.client.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &stss); err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, fmt.Errorf("list statefulsets with namespace %s failed: %s", namespace, err.Error())
		}
	}

	for _, sts := range stss.Items {
		if isMeshedWorkload(sts.Spec.Template) {
			resources = append(resources, types.Resource{
				Type: StatefulSet,
				Name: sts.Name,
			})
		}
	}

	return resources, nil
}

func isMeshedWorkload(spec corev1.PodTemplateSpec) bool {
	if enabled, ok := spec.Annotations[LinkerdInjectAnnotation]; ok && enabled == "enabled" {
		return true
	}

	for _, c := range spec.Spec.Containers {
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

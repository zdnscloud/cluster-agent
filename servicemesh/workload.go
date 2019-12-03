package servicemesh

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

type WorkloadManager struct {
	apiServerURL *url.URL
	client       client.Client
}

func newWorkloadManager(apiServerURL *url.URL, cli client.Client) *WorkloadManager {
	return &WorkloadManager{
		apiServerURL: apiServerURL,
		client:       cli,
	}
}

func (m *WorkloadManager) Get(ctx *resource.Context) resource.Resource {
	namespace := ctx.Resource.GetParent().GetParent().GetID()
	id := ctx.Resource.(*types.Workload).GetID()
	workload, err := m.getWorkload(namespace, id)
	if err != nil {
		log.Warnf("get workload %s failed: %s", id, err.Error())
		return nil
	}

	return workload
}

func (m *WorkloadManager) getWorkload(namespace, id string) (*types.Workload, error) {
	resourceType, resourceName, err := getResourceTypeAndName(id)
	if err != nil {
		return nil, err
	}

	stat, err := getStat(m.apiServerURL, namespace, resourceType, resourceName)
	if err != nil {
		return nil, fmt.Errorf("get workload %s/%s stats with namespace %s failed: %s",
			resourceType, resourceName, namespace, err.Error())
	}

	inbound, err := getStatsTo(m.apiServerURL, namespace, resourceType, resourceName)
	if err != nil {
		return nil, fmt.Errorf("get workload %s/%s inbound stats with namespace %s failed: %s",
			resourceType, resourceName, namespace, err.Error())
	}

	outbound, err := getStatsFrom(m.apiServerURL, namespace, resourceType, resourceName)
	if err != nil {
		return nil, fmt.Errorf("get workload %s/%s outbound stats with namespace %s failed: %s",
			resourceType, resourceName, namespace, err.Error())
	}

	pods, err := getPods(m.client, m.apiServerURL, namespace, resourceType, resourceName)
	if err != nil {
		return nil, err
	}

	workload := &types.Workload{
		Stat:     stat,
		Inbound:  inbound,
		Outbound: outbound,
		Pods:     pods,
	}

	workload.SetID(id)
	return workload, nil
}

func getResourceTypeAndName(id string) (string, string, error) {
	if len(id) <= 3 {
		return "", "", fmt.Errorf("invalid workload id %s, its len must be longer than 3", id)
	}

	prefix := id[:3]
	name := id[3:]
	var typ string
	switch prefix {
	case DeploymentPrefix:
		typ = Deployment
	case DaemonSetPrefix:
		typ = DaemonSet
	case StatefulSetPrefix:
		typ = StatefulSetPrefix
	default:
		return "", "", fmt.Errorf("unspported workload prefix %s", prefix)
	}

	return typ, name, nil
}

func getPods(cli client.Client, apiServerURL *url.URL, namespace, resourceType, resourceName string) (types.Pods, error) {
	selector, err := getPodParentSelector(cli, namespace, resourceType, resourceName)
	if err != nil {
		return nil, err
	}

	podList := corev1.PodList{}
	if err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace, LabelSelector: selector}, &podList); err != nil {
		if apierrors.IsNotFound(err) == false {
			return nil, fmt.Errorf("list %s/%s pods with namespace %s failed: %s", resourceType, resourceName, namespace, err.Error())
		}
	}

	var pods types.Pods
	for _, p := range podList.Items {
		stat, err := getStat(apiServerURL, namespace, Pod, p.Name)
		if err != nil {
			return nil, err
		}

		pod := &types.Pod{Stat: stat}
		pod.SetID(p.Name)
		pods = append(pods, pod)
	}

	sort.Sort(pods)
	return pods, nil
}

func getPodParentSelector(cli client.Client, namespace, resourceType, resourceName string) (labels.Selector, error) {
	var selector *metav1.LabelSelector
	switch resourceType {
	case Deployment:
		deploy := appsv1.Deployment{}
		if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, resourceName}, &deploy); err != nil {
			return nil, fmt.Errorf("get deployment %s with namespace %s failed: %s", resourceName, namespace, err.Error())
		}

		selector = deploy.Spec.Selector
	case DaemonSet:
		ds := appsv1.DaemonSet{}
		if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, resourceName}, &ds); err != nil {
			return nil, fmt.Errorf("get daemonset %s with namespace %s failed: %s", resourceName, namespace, err.Error())
		}

		selector = ds.Spec.Selector
	case StatefulSet:
		sts := appsv1.StatefulSet{}
		if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, resourceName}, &sts); err != nil {
			return nil, fmt.Errorf("get statefulset %s with namespace %s failed: %s", resourceName, namespace, err.Error())
		}

		selector = sts.Spec.Selector
	}

	if selector == nil {
		return nil, fmt.Errorf("workload %s/%s with namespace %s no selector", resourceType, resourceName, namespace)
	}

	return metav1.LabelSelectorAsSelector(selector)
}

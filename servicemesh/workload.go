package servicemesh

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

type WorkloadManager struct {
	apiServerURL *url.URL
	groupManager *WorkloadGroupManager
}

func newWorkloadManager(apiServerURL *url.URL, groupManager *WorkloadGroupManager) *WorkloadManager {
	return &WorkloadManager{
		apiServerURL: apiServerURL,
		groupManager: groupManager,
	}
}

func (m *WorkloadManager) Get(ctx *resource.Context) resource.Resource {
	namespace := ctx.Resource.GetParent().GetParent().GetID()
	id := ctx.Resource.(*types.Workload).GetID()
	workload, err := m.getWorkload(namespace, id)
	if err != nil {
		log.Warnf("get workload with id %s failed: %s", id, err.Error())
		return nil
	}

	return workload
}

func (m *WorkloadManager) getWorkload(namespace, id string) (*types.Workload, error) {
	statOptions, err := m.getStatOptions(namespace, id)
	if err != nil {
		return nil, err
	}

	resultCh, err := errgroup.Batch(statOptions, func(options interface{}) (interface{}, error) {
		return getWorkloadWithOptions(options.(*StatOptions))
	})
	if err != nil {
		return nil, err
	}

	workload := &types.Workload{}
	for result := range resultCh {
		w := result.(*types.Workload)
		if len(w.Inbound) != 0 {
			workload.Inbound = w.Inbound
		} else if len(w.Outbound) != 0 {
			workload.Outbound = w.Outbound
		} else if w.Stat.Resource.Type == Pod {
			pod := &types.Pod{Stat: w.Stat}
			pod.SetID(pod.Stat.Resource.Name)
			workload.Pods = append(workload.Pods, pod)
		} else {
			workload.Stat = w.Stat
		}
	}

	sort.Sort(workload.Pods)
	workload.SetID(id)
	return workload, nil
}

func (m *WorkloadManager) getStatOptions(namespace, id string) ([]*StatOptions, error) {
	resourceType, resourceName, err := getResourceTypeAndName(id)
	if err != nil {
		return nil, err
	}

	options := genBasicStatOptions(m.apiServerURL, namespace, resourceType, resourceName)
	pods, err := m.groupManager.GetWorkloadPods(namespace, id)
	if err != nil {
		return nil, err
	}

	for _, podName := range pods {
		options = append(options, &StatOptions{
			ApiServerURL: m.apiServerURL,
			Namespace:    namespace,
			ResourceType: Pod,
			ResourceName: podName,
		})
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
		typ = Deployment
	case DaemonSetPrefix:
		typ = DaemonSet
	case StatefulSetPrefix:
		typ = StatefulSet
	default:
		return "", "", fmt.Errorf("unspported workload prefix %s", prefix)
	}

	return typ, name, nil
}

func genBasicStatOptions(apiServerURL *url.URL, namespace, resourceType, resourceName string) []*StatOptions {
	return []*StatOptions{
		&StatOptions{
			ApiServerURL: apiServerURL,
			Namespace:    namespace,
			ResourceType: resourceType,
			ResourceName: resourceName,
		},
		&StatOptions{
			ApiServerURL: apiServerURL,
			Namespace:    namespace,
			ResourceType: resourceType,
			ResourceName: resourceName,
			From:         true,
		},
		&StatOptions{
			ApiServerURL: apiServerURL,
			Namespace:    namespace,
			ResourceType: resourceType,
			ResourceName: resourceName,
			To:           true,
		},
	}
}

func getWorkloadWithOptions(opts *StatOptions) (*types.Workload, error) {
	if opts.From {
		stats, err := getStats(opts)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s outbound stats with namespace %s failed: %s",
				opts.ResourceType, opts.ResourceName, opts.Namespace, err.Error())
		}

		return &types.Workload{Outbound: stats}, nil
	} else if opts.To {
		stats, err := getStats(opts)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s inbound stats with namespace %s failed: %s",
				opts.ResourceType, opts.ResourceName, opts.Namespace, err.Error())
		}
		return &types.Workload{Inbound: stats}, nil
	} else {
		stat, err := getStat(opts)
		if err != nil {
			return nil, fmt.Errorf("get %s/%s stats with namespace %s failed: %s",
				opts.ResourceType, opts.ResourceName, opts.Namespace, err.Error())
		}
		return &types.Workload{Stat: stat}, nil
	}
}

package servicemesh

import (
	"net/url"

	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

const (
	Pod = "pod"
)

type PodManager struct {
	apiServerURL *url.URL
	groupManager *WorkloadGroupManager
}

func newPodManager(apiServerURL *url.URL, groupManager *WorkloadGroupManager) *PodManager {
	return &PodManager{
		apiServerURL: apiServerURL,
		groupManager: groupManager,
	}
}

func (m *PodManager) Get(ctx *resource.Context) resource.Resource {
	namespace := ctx.Resource.GetParent().GetParent().GetParent().GetID()
	workloadId := ctx.Resource.GetParent().GetID()
	podId := ctx.Resource.(*types.Pod).GetID()
	pod, err := m.getPod(namespace, workloadId, podId)
	if err != nil {
		log.Warnf("get pod %s failed: %s", podId, err.Error())
		return nil
	}

	return pod
}

func (m *PodManager) getPod(namespace, workloadId, podName string) (*types.Pod, error) {
	if err := m.groupManager.IsPodBelongToWorkload(namespace, workloadId, podName); err != nil {
		return nil, err
	}

	resultCh, err := errgroup.Batch(genBasicStatOptions(m.apiServerURL, namespace, Pod, podName),
		func(options interface{}) (interface{}, error) {
			return getWorkloadWithOptions(options.(*StatOptions))
		},
	)
	if err != nil {
		return nil, err
	}

	pod := &types.Pod{}
	for result := range resultCh {
		p := result.(*types.Workload)
		if len(p.Inbound) != 0 {
			pod.Inbound = p.Inbound
		} else if len(p.Outbound) != 0 {
			pod.Outbound = p.Outbound
		} else {
			pod.Stat = p.Stat
		}
	}

	pod.SetID(podName)
	return pod, nil
}

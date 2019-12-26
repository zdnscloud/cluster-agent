package servicemesh

import (
	"net/url"

	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

const (
	ResourceTypePod = "pod"
)

type PodManager struct {
	apiServerURL    *url.URL
	workloadManager *WorkloadManager
}

func newPodManager(apiServerURL *url.URL, workloadManager *WorkloadManager) *PodManager {
	return &PodManager{
		apiServerURL:    apiServerURL,
		workloadManager: workloadManager,
	}
}

func (m *PodManager) Get(ctx *resource.Context) resource.Resource {
	namespace := ctx.Resource.GetParent().GetParent().GetID()
	workloadId := ctx.Resource.GetParent().GetID()
	podId := ctx.Resource.(*types.SvcMeshPod).GetID()
	pod, err := m.getPod(namespace, workloadId, podId)
	if err != nil {
		log.Warnf("get pod %s failed: %s", podId, err.Error())
		return nil
	}

	return pod
}

func (m *PodManager) getPod(namespace, workloadId, podName string) (*types.SvcMeshPod, error) {
	podOwners, err := m.workloadManager.GetPodOwners(namespace, workloadId, podName)
	if err != nil {
		return nil, err
	}

	resultCh, err := errgroup.Batch(genBasicStatOptions(m.apiServerURL, namespace, ResourceTypePod, podName),
		func(option interface{}) (interface{}, error) {
			return getWorkloadWithOption(option.(*StatOption))
		},
	)
	if err != nil {
		return nil, err
	}

	pod := &types.SvcMeshPod{}
	for result := range resultCh {
		switch r := result.(*StatResult); r.RequestType {
		case RequestTypeInbound:
			pod.Inbound = m.statResultBoundToPodBound(podOwners, namespace, r.Inbound)
		case RequestTypeOutbound:
			pod.Outbound = m.statResultBoundToPodBound(podOwners, namespace, r.Outbound)
		case RequestTypePod:
			pod.Stat = r.Stat
		}
	}

	pod.SetID(podName)
	return pod, nil
}

func (m *PodManager) statResultBoundToPodBound(podOwners map[string]string, namespace string, stats types.Stats) types.Stats {
	var ss types.Stats
	for _, s := range stats {
		s.WorkloadID = podOwners[s.Resource.Name]
		ss = append(ss, s)
	}

	return ss
}

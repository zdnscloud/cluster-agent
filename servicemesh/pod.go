package servicemesh

import (
	"fmt"
	"net/url"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

const (
	Pod = "pod"
)

type PodManager struct {
	apiServerURL *url.URL
}

func newPodManager(apiServerURL *url.URL) *PodManager {
	return &PodManager{apiServerURL}
}

func (m *PodManager) Get(ctx *resource.Context) resource.Resource {
	namespace := ctx.Resource.GetParent().GetParent().GetParent().GetID()
	podId := ctx.Resource.(*types.Pod).GetID()
	pod, err := m.getPod(namespace, podId)
	if err != nil {
		log.Warnf("get pod %s stat with namespace %s failed: %s", podId, namespace, err.Error())
		return nil
	}

	return pod
}

func (m *PodManager) getPod(namespace, name string) (*types.Pod, error) {
	stat, err := getStat(m.apiServerURL, namespace, Pod, name)
	if err != nil {
		return nil, err
	}

	inbound, err := getStatsTo(m.apiServerURL, namespace, Pod, name)
	if err != nil {
		return nil, fmt.Errorf("get pod %s inbound stats with namespace %s failed: %s", name, namespace, err.Error())
	}

	outbound, err := getStatsFrom(m.apiServerURL, namespace, Pod, name)
	if err != nil {
		return nil, fmt.Errorf("get pod %s outbound stats with namespace %s failed: %s", name, namespace, err.Error())
	}

	pod := &types.Pod{
		Stat:     stat,
		Inbound:  inbound,
		Outbound: outbound,
	}

	pod.SetID(name)
	return pod, nil
}

package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type WorkloadPod struct {
	resource.ResourceBase `json:",inline"`
	Stat                  Stat  `json:"stat,omitempty"`
	Inbound               Stats `json:"inbound,omitempty"`
	Outbound              Stats `json:"outbound,omitempty"`
}

func (p WorkloadPod) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Workload{}}
}

type Pods []*WorkloadPod

func (p Pods) Len() int {
	return len(p)
}

func (p Pods) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Pods) Less(i, j int) bool {
	return p[i].Stat.Resource.Name < p[j].Stat.Resource.Name
}

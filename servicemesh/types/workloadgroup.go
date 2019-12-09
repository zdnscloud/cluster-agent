package types

import (
	"github.com/zdnscloud/cluster-agent/service"
	"github.com/zdnscloud/gorest/resource"
)

type WorkloadGroup struct {
	resource.ResourceBase `json:",inline"`
	Workloads             Workloads `json:"workloads,omitempty"`
}

func (w WorkloadGroup) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{service.Namespace{}}
}

type WorkloadGroups []*WorkloadGroup

func (w WorkloadGroups) Len() int {
	return len(w)
}

func (w WorkloadGroups) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w WorkloadGroups) Less(i, j int) bool {
	return len(w[j].Workloads) < len(w[i].Workloads)
}

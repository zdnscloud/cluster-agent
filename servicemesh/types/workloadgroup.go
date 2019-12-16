package types

import (
	"github.com/zdnscloud/cluster-agent/service"
	"github.com/zdnscloud/gorest/resource"
)

type SvcMeshWorkloadGroup struct {
	resource.ResourceBase `json:",inline"`
	Workloads             Workloads `json:"workloads,omitempty"`
}

func (w SvcMeshWorkloadGroup) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{service.Namespace{}}
}

type SvcMeshWorkloadGroups []*SvcMeshWorkloadGroup

func (w SvcMeshWorkloadGroups) Len() int {
	return len(w)
}

func (w SvcMeshWorkloadGroups) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w SvcMeshWorkloadGroups) Less(i, j int) bool {
	if len(w[j].Workloads) == len(w[i].Workloads) {
		return w[i].GetID() < w[j].GetID()
	}

	return len(w[j].Workloads) < len(w[i].Workloads)
}

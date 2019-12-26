package types

import (
	"github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/cluster-agent/service"
)

type SvcMeshWorkload struct {
	resource.ResourceBase `json:",inline"`
	GroupID               string   `json:"groupId,omitempty"`
	Destinations          []string `json:"destinations,omitempty"`
	Stat                  Stat     `json:"stat,omitempty"`
	Inbound               Stats    `json:"inbound,omitempty"`
	Outbound              Stats    `json:"outbound,omitempty"`
	Pods                  Pods     `json:"pods,omitempty"`
}

func (w SvcMeshWorkload) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{service.Namespace{}}
}

type SvcMeshWorkloads []*SvcMeshWorkload

func (w SvcMeshWorkloads) Len() int {
	return len(w)
}

func (w SvcMeshWorkloads) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w SvcMeshWorkloads) Less(i, j int) bool {
	return w[i].Stat.ID < w[j].Stat.ID
}

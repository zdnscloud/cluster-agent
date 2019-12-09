package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Workload struct {
	resource.ResourceBase `json:",inline"`
	Destinations          []string `json:"destinations,omitempty"`
	Stat                  Stat     `json:"stat,omitempty"`
	Inbound               Stats    `json:"inbound,omitempty"`
	Outbound              Stats    `json:"outbound,omitempty"`
	Pods                  Pods     `json:"pods,omitempty"`
}

func (w Workload) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{WorkloadGroup{}}
}

type Workloads []*Workload

func (w Workloads) Len() int {
	return len(w)
}

func (w Workloads) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

func (w Workloads) Less(i, j int) bool {
	if w[i].Stat.Resource.Type == w[j].Stat.Resource.Type {
		return w[i].Stat.Resource.Name < w[j].Stat.Resource.Name
	}

	return w[i].Stat.Resource.Type < w[j].Stat.Resource.Type
}

package service

import (
	"github.com/zdnscloud/gorest/resource"

	common "github.com/zdnscloud/cluster-agent/commonresource"
)

type InnerService struct {
	resource.ResourceBase `json:",inline"`
	Name                  string      `json:"name"`
	Workloads             []*Workload `json:"workloads"`
}

func (s InnerService) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{common.Namespace{}}
}

type Workload struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Pods []Pod  `json:"pods"`
}

type Pod struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type OuterService struct {
	resource.ResourceBase `json:",inline"`
	EntryPoint            string                  `json:"entryPoint"`
	Services              map[string]InnerService `json:"services"`
}

func (s OuterService) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{common.Namespace{}}
}

type InnerServiceByName []*InnerService

func (a InnerServiceByName) Len() int           { return len(a) }
func (a InnerServiceByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a InnerServiceByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type OuterServiceByEntryPoint []*OuterService

func (a OuterServiceByEntryPoint) Len() int           { return len(a) }
func (a OuterServiceByEntryPoint) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a OuterServiceByEntryPoint) Less(i, j int) bool { return a[i].EntryPoint < a[j].EntryPoint }

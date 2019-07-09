package service

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetNamespaceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
}

type Namespace struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
}

var NamespaceType = resttypes.GetResourceType(Namespace{})

func SetInnerServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	//schema.Parent = NamespaceType
	schema.Parents = []string{NamespaceType}
}

type InnerService struct {
	resttypes.Resource `json:",inline"`
	Name               string      `json:"name"`
	Workloads          []*Workload `json:"workloads"`
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

var InnerServiceType = resttypes.GetResourceType(InnerService{})

type OuterService struct {
	resttypes.Resource `json:",inline"`
	EntryPoint         string                  `json:"entryPoint"`
	Services           map[string]InnerService `json:"services"`
}

func SetOuterServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	//schema.Parent = NamespaceType
	schema.Parents = []string{NamespaceType}
}

var OuterServiceType = resttypes.GetResourceType(OuterService{})

type InnerServiceByName []InnerService

func (a InnerServiceByName) Len() int           { return len(a) }
func (a InnerServiceByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a InnerServiceByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type OuterServiceByEntryPoint []OuterService

func (a OuterServiceByEntryPoint) Len() int           { return len(a) }
func (a OuterServiceByEntryPoint) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a OuterServiceByEntryPoint) Less(i, j int) bool { return a[i].EntryPoint < a[j].EntryPoint }

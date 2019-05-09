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
	schema.Parent = NamespaceType
}

type InnerService struct {
	resttypes.Resource `json:",inline"`
	Name               string     `json:"name"`
	Workloads          []Workload `json:"workloads"`
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
	Domain             string                  `json:"domain"`
	Port               int                     `json:"port"`
	Services           map[string]InnerService `json:"services"`
}

func SetOuterServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.Parent = NamespaceType
}

var OuterServiceType = resttypes.GetResourceType(OuterService{})

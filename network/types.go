package network

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetNodeSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
}

type Node struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	IP                 string `json:"ip"`
}

var NodeType = resttypes.GetResourceType(Node{})

func SetPodSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
}

type Pod struct {
	resttypes.Resource `json:",inline"`
	NodeName           string  `json:"nodeName"`
	PodCIDR            string  `json:"podCIDR"`
	PodIPs             []PodIP `json:"podIPs,omitempty"`
}

type PodIP struct {
	Namespace string `json:"-"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
}

var PodType = resttypes.GetResourceType(Pod{})

func SetServiceSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
}

type Service struct {
	resttypes.Resource `json:",inline"`
	Namespace          string `json:"-"`
	Name               string `json:"name"`
	IP                 string `json:"ip"`
}

var ServiceType = resttypes.GetResourceType(Service{})

package network

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetNodeNetworkSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
}

type NodeNetwork struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	IP                 string `json:"ip"`
}

var NodeNetworkType = resttypes.GetResourceType(NodeNetwork{})

func SetPodNetworkSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
}

type PodNetwork struct {
	resttypes.Resource `json:",inline"`
	NodeName           string  `json:"nodeName"`
	PodCIDR            string  `json:"podCIDR"`
	PodIPs             []PodIP `json:"networks,omitempty"`
}

type PodIP struct {
	Namespace string `json:"-"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
}

var PodNetworkType = resttypes.GetResourceType(PodNetwork{})

func SetServiceNetworkSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
}

type ServiceNetwork struct {
	resttypes.Resource `json:",inline"`
	Namespace          string `json:"-"`
	Name               string `json:"name"`
	IP                 string `json:"ip"`
}

var ServiceNetworkType = resttypes.GetResourceType(ServiceNetwork{})

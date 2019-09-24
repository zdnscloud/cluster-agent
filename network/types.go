package network

import (
	resttypes "github.com/zdnscloud/gorest/resource"
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

type NodeNetworks []NodeNetwork

func (n NodeNetworks) Len() int {
	return len(n)
}
func (n NodeNetworks) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}
func (n NodeNetworks) Less(i, j int) bool {
	return n[i].Name < n[j].Name
}

func SetPodNetworkSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
}

type PodNetwork struct {
	resttypes.Resource `json:",inline"`
	NodeName           string  `json:"nodeName"`
	PodCIDR            string  `json:"podCIDR"`
	PodIPs             []PodIP `json:"podIPs"`
}

type PodIP struct {
	Namespace string `json:"-"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
}

var PodNetworkType = resttypes.GetResourceType(PodNetwork{})

type PodNetworks []PodNetwork

func (p PodNetworks) Len() int {
	return len(p)
}
func (p PodNetworks) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p PodNetworks) Less(i, j int) bool {
	return p[i].NodeName < p[j].NodeName
}

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

type ServiceNetworks []ServiceNetwork

func (s ServiceNetworks) Len() int {
	return len(s)
}
func (s ServiceNetworks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ServiceNetworks) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

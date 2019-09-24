package network

import (
	"github.com/zdnscloud/gorest/resource"
)

type NodeNetwork struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	IP                    string `json:"ip"`
}

type NodeNetworks []*NodeNetwork

func (n NodeNetworks) Len() int {
	return len(n)
}
func (n NodeNetworks) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}
func (n NodeNetworks) Less(i, j int) bool {
	return n[i].Name < n[j].Name
}

type PodNetwork struct {
	resource.ResourceBase `json:",inline"`
	NodeName              string  `json:"nodeName"`
	PodCIDR               string  `json:"podCIDR"`
	PodIPs                []PodIP `json:"podIPs"`
}

type PodIP struct {
	Namespace string `json:"-"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
}

type PodNetworks []*PodNetwork

func (p PodNetworks) Len() int {
	return len(p)
}
func (p PodNetworks) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p PodNetworks) Less(i, j int) bool {
	return p[i].NodeName < p[j].NodeName
}

type ServiceNetwork struct {
	resource.ResourceBase `json:",inline"`
	Namespace             string `json:"-"`
	Name                  string `json:"name"`
	IP                    string `json:"ip"`
}

type ServiceNetworks []*ServiceNetwork

func (s ServiceNetworks) Len() int {
	return len(s)
}
func (s ServiceNetworks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ServiceNetworks) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

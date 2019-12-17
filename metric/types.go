package metric

import (
	"github.com/zdnscloud/gorest/resource"

	common "github.com/zdnscloud/cluster-agent/commonresource"
)

type Workload struct {
	Type       string `json:"type,omitempty"`
	Name       string `json:"name,omitempty"`
	MetricPort int    `json:"metricPort,omitempty"`
	MetricPath string `json:"metricPort,omitempty"`
	Pods       []Pod  `json:"pods,omitempty"`
}

type Pod struct {
	Name string `json:"name,omitempty"`
	IP   string `json:"ip,omitempty"`
}

type Metric struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"metrics,omitempty"`
}

func (m Metric) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{common.DaemonSet{}, common.Deployment{}, common.StatefulSet{}}
}

type Metrics []*Metric

func (m Metrics) Len() int {
	return len(m)
}

func (m Metrics) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m Metrics) Less(i, j int) bool {
	return m[i].Name < m[j].Name
}

package commonresource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Namespace struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name,omitempty"`
}

type Deployment struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name,omitempty"`
}

func (d Deployment) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

type DaemonSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name,omitempty"`
}

func (d DaemonSet) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

type StatefulSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name,omitempty"`
}

func (s StatefulSet) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

var (
	ResourceTypeDeployment  = resource.DefaultKindName(Deployment{})
	ResourceTypeDaemonSet   = resource.DefaultKindName(DaemonSet{})
	ResourceTypeStatefulSet = resource.DefaultKindName(StatefulSet{})
)

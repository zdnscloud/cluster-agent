package nodeagent

import (
	"github.com/zdnscloud/gorest/resource"
)

type NodeAgent struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	Address               string `json:"address"`
}

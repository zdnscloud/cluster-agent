package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Storage struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	PVs                   []PV   `json:"pvs"`
}

type PV struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	UsedSize         string `json:"usedSize"`
	FreeSize         string `json:"freeSize"`
	Pods             []Pod  `json:"pods"`
	StorageClassName string `json:"-"`
	Node             string `json:"node"`
}

type Pod struct {
	Name string `json:"name"`
}

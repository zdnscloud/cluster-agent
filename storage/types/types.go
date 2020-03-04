package types

import (
	"github.com/zdnscloud/gorest/resource"
)

type Storage struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	PVs                   []*PV  `json:"pvs"`
}

type PV struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	UsedSize         string `json:"usedSize"`
	FreeSize         string `json:"freeSize"`
	Pods             []Pod  `json:"pods"`
	StorageClassName string `json:"-"`
	Node             string `json:"node"`
	PVC              string `json:"pvc"`
}

type Pod struct {
	Name string `json:"name"`
}

type Pvs []*PV

func (s Pvs) Len() int           { return len(s) }
func (s Pvs) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Pvs) Less(i, j int) bool { return s[i].PVC < s[j].PVC }

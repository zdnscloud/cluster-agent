package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

type StorageInfo struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
	TotalSize          int    `json:"totalsize,omitempty"`
	FreeSize           int    `json:"freesize,omitempty"`
	Nodes              []Node `json:"nodes,omitempty"`
	PVs                []PV   `json:"pvs,omitempty"`
}

type PV struct {
	Name string
	Size int
	Pods []Pod
}

type Node struct {
	Name     string
	Addr     string
	Size     uint64
	FreeSize uint64
}

type Pod struct {
	Name      string
	Namespace string
}

type Pvc struct {
	Name         string
	Namespace    string
	StorageClass string
	VolumeName   string
	Pods         []Pod
}

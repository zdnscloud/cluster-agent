package lvm

import (
	"github.com/zdnscloud/cluster-agent/storage/types"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	CtrlName = "lvmcontroller"
)

func SetStorageSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

var StorageType = resttypes.GetResourceType(Storage{})

type Storage struct {
	resttypes.Resource `json:",inline"`
	Name               string                 `json:"name,omitempty"`
	TotalSize          int                    `json:"totalsize,omitempty"`
	FreeSize           int                    `json:"freesize,omitempty"`
	Nodes              []types.Node           `json:"nodes,omitempty"`
	PVs                []types.PV             `json:"pvs,omitempty"`
	PvAndPvc           map[string]types.Pvc   `json:"_"`
	PvcAndPod          map[string][]types.Pod `json:"_"`
}

/*
type Node struct {
	Name     string
	Addr     string
	Size     uint64
	FreeSize uint64
}

type VG struct {
	Name     string   `json:"name,omitempty"`
	Size     int      `json:"size,omitempty"`
	FreeSize int      `json:"free_size,omitempty"`
	Uuid     string   `json:"uuid,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type PV struct {
	Name   string `json:"name,omitempty"`
	Size   int    `json:"size,omitempty"`
	Pods   []Pod  `json:"pods,omitempty"`
	Pvc    string `json:"_"`
	Status string `json:"_"`
}

type Pvc struct {
	Name         string
	Namespace    string
	StorageClass string
	VolumeName   string
	Pods         []Pod
}

type Pod struct {
	Name      string
	Namespace string
}*/

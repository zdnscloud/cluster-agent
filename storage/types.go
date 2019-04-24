package storage

import (
	resttypes "github.com/zdnscloud/gorest/types"
	corev1 "k8s.io/api/core/v1"
)

func SetStorageSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

var StorageType = resttypes.GetResourceType(Storage{})

type Storage struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
	TotalSize          int    `json:"totalsize,omitempty"`
	FreeSize           int    `json:"freesize,omitempty"`
	Nodes              []Node `json:"nodes,omitempty"`
	PVs                []Pv   `json:"pvs,omitempty"`
}

type Node struct {
	Name      string `json:"name,omitempty"`
	Addr      string `json:"addr,omitempty"`
	TotalSize int    `json:"totalsize,omitempty"`
	FreeSize  int    `json:"freesize,omitempty"`
	Vgs       []VG   `json:"vgs,omitempty"`
}

type VG struct {
	Name     string   `json:"name,omitempty"`
	Size     int      `json:"size,omitempty"`
	FreeSize int      `json:"free_size,omitempty"`
	Uuid     string   `json:"uuid,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type Pv struct {
	Name   string                       `json:"name,omitempty"`
	Size   int                          `json:"size,omitempty"`
	Pods   []string                     `json:"pods,omitempty"`
	Pvc    string                       `json:"pvc,omitempty"`
	Status corev1.PersistentVolumePhase `json:"status,omitempty"`
}

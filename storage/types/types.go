package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
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
	Size               int    `json:"size,omitempty"`
	FreeSize           int    `json:"freesize,omitempty"`
	Nodes              []Node `json:"nodes,omitempty"`
	PVs                []PV   `json:"pvs,omitempty"`
}

type PV struct {
	Name             string `json:"name,omitempty"`
	Size             int    `json:"size,omitempty"`
	Pods             []Pod  `json:"pods,omitempty"`
	StorageClassName string `json:"-"`
}

type Node struct {
	Name     string `json:"name,omitempty"`
	Size     int    `json:"size,omitempty"`
	FreeSize int    `json:"freesize,omitempty"`
}

type Pod struct {
	Name string `json:"name,omitempty"`
}

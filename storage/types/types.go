package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetStorageSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

var StorageType = resttypes.GetResourceType(Storage{})

type Storage struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name"`
	PVs                []PV   `json:"pvs"`
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

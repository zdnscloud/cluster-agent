package handler

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetStorageSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

func SetNodeSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = StorageType
}

func SetVGSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = NodeType
}
func SetLVMSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = VGType
}

var StorageType = resttypes.GetResourceType(Storage{})
var NodeType = resttypes.GetResourceType(Node{})
var VGType = resttypes.GetResourceType(VG{})
var LVMType = resttypes.GetResourceType(LVM{})

type Storage struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
	TotalSize          uint64 `json:"totalsize,omitempty"`
	FreeSize           uint64 `json:"freesize,omitempty"`
	//StorageNode        []Node `json:"storagenode,omitempty"`
}

type Node struct {
	resttypes.Resource `json:",inline"`
	Name               string `json:"name,omitempty"`
	Addr               string `json:"addr,omitempty"`
	TotalSize          uint64 `json:"totalsize,omitempty"`
	FreeSize           uint64 `json:"freesize,omitempty"`
	//Vgs                []VG   `json:"vgs,omitempty"`
}

type VG struct {
	resttypes.Resource `json:",inline"`
	Name               string   `json:"name,omitempty"`
	Size               uint64   `json:"size,omitempty"`
	FreeSize           uint64   `json:"free_size,omitempty"`
	Uuid               string   `json:"uuid,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	//Lvms               []LVM    `json:"lvms,omitempty"`
}

type LVM struct {
	resttypes.Resource `json:",inline"`
	Name               string   `json:"name,omitempty"`
	Size               uint64   `json:"size,omitempty"`
	Uuid               string   `json:"uuid,omitempty"`
	Tags               []string `json:"tags,omitempty"`
}

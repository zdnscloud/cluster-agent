package blockinfo

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetBlockInfoSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
}

var BlockInfoType = resttypes.GetResourceType(BlockInfo{})

type BlockInfo struct {
	resttypes.Resource `json:",inline"`
	Hosts              []Host `json:"hosts"`
}

type Host struct {
	NodeName     string `json:"nodeName"`
	BlockDevices []Dev  `json:"blockDevices"`
}
type Dev struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	Parted     bool   `json:"parted"`
	Filesystem bool   `json:"filesystem"`
	Mount      bool   `json:"mount"`
}

package storageclass

import (
	"github.com/zdnscloud/cluster-agent/storage/types"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	CtrlName      = "storagecontroller"
	ZKENFSPvcName = "nfs-data-nfs-provisioner-0"
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
	Nodes              []types.Node           `json:"nodes,omitempty"`
	PVs                []types.PV             `json:"pvs,omitempty"`
	PvAndPvc           map[string]types.Pvc   `json:"_"`
	PvcAndPod          map[string][]types.Pod `json:"_"`
}

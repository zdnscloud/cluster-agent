package handler

import (
	//"fmt"
	resttypes "github.com/zdnscloud/gorest/types"
)

type VGManager struct {
	DefaultHandler
	storages *StorageManager
}

func newVGManager(storages *StorageManager) *VGManager {
	return &VGManager{storages: storages}
}

func (m *VGManager) List(ctx *resttypes.Context) interface{} {
	id := ctx.Object.GetParent().GetID()
	addr := nameToAddr(id)
	vgs := getVgSlice(addr)
	return vgs
}

func (m *VGManager) Get(ctx *resttypes.Context) interface{} {
	nodename := ctx.Object.GetParent().GetID()
	id := ctx.Object.GetID()

	addr := nameToAddr(nodename)
	vgstmp := getVgSlice(addr)
	var vgs []VG
	for _, v := range vgstmp {
		if id == v.Name {
			vgs = append(vgs, v)
		}
	}
	return vgs
}

package handler

import (
	//"fmt"
	resttypes "github.com/zdnscloud/gorest/types"
)

type LVMManager struct {
	DefaultHandler
	storages *StorageManager
}

func newLVMManager(storages *StorageManager) *LVMManager {
	return &LVMManager{storages: storages}
}

func (m *LVMManager) List(ctx *resttypes.Context) interface{} {
	vgname := ctx.Object.GetParent().GetID()
	nodename := ctx.Object.GetParent().GetParent().GetID()
	addr := nameToAddr(nodename)
	return getLvmSlice(addr, vgname)
}

func (m *LVMManager) Get(ctx *resttypes.Context) interface{} {
	vgname := ctx.Object.GetParent().GetID()
	nodename := ctx.Object.GetParent().GetParent().GetID()

	addr := nameToAddr(nodename)
	vgs := getLvmSlice(addr, vgname)
	id := ctx.Object.GetID()
	var lvms []LVM
	for _, v := range vgs {
		if id == v.Name {
			lvms = append(lvms, v)
		}
	}
	return lvms
}

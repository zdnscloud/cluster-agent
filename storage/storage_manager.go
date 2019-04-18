package storage

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

type StorageManager struct {
	DefaultHandler
	storages []Storage
}

func newStorageManager() *StorageManager {
	return &StorageManager{}
}

func (m *StorageManager) List(ctx *resttypes.Context) interface{} {
	return getStorages()
}

func (m *StorageManager) Get(ctx *resttypes.Context) interface{} {
	var storages []Storage
	id := ctx.Object.GetID()
	storageclasss := getStorages()
	for _, s := range storageclasss {
		if id == s.Name {
			storages = append(storages, s)
		}
	}
	return storages
}

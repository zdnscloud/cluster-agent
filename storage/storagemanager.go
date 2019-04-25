package storage

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/storage/storageclass"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "storage.zcloud.cn",
	}
)

type Storage interface {
	GetStorageClass() string
	GetStroageInfo(string) types.StorageInfo
}

type StorageManager struct {
	api.DefaultHandler
	storages []Storage
}

func RegisterHandler(router gin.IRoutes, cache cache.Cache) error {
	schemas := resttypes.NewSchemas()
	m := newStorageManager(cache)
	schemas.MustImportAndCustomize(&Version, storageclass.Storage{}, m, storageclass.SetStorageSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())
	return nil
}

func newStorageManager(c cache.Cache) *StorageManager {
	lvm, err := storageclass.New(c, "lvm")
	if err != nil {
		log.Errorf("Init LVM Storage falied:%s", err.Error())
	}
	nfs, err := storageclass.New(c, "nfs")
	if err != nil {
		log.Errorf("Init NFS Storage falied:%s", err.Error())
	}
	m := &StorageManager{
		storages: []Storage{lvm, nfs},
	}
	return m
}

func (m *StorageManager) Get(ctx *resttypes.Context) interface{} {
	cls := ctx.Object.GetID()
	fmt.Println(cls)
	for _, s := range m.storages {
		if s.GetStorageClass() == cls {
			return s.GetStroageInfo(cls)
		}
	}
	return nil
}

func (m *StorageManager) List(ctx *resttypes.Context) interface{} {
	var infos []types.StorageInfo
	for _, s := range m.storages {
		cls := s.GetStorageClass()
		info := s.GetStroageInfo(cls)
		infos = append(infos, info)
	}
	return infos
}

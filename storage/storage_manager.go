package storage

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cluster-agent/storage/lvm"
	"github.com/zdnscloud/cluster-agent/storage/nfs"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	LogLevel = "debug"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "storage.zcloud.cn",
	}
)

type Storage interface {
	GetStorageClass() string
	GetStroageInfo() types.StorageInfo
}

type StorageManager struct {
	api.DefaultHandler
	storages []Storage
}

func RegisterHandler(router gin.IRoutes, cache cache.Cache) error {
	schemas := resttypes.NewSchemas()
	m, _ := newStorageManager(cache)
	schemas.MustImportAndCustomize(&Version, lvm.Storage{}, m, lvm.SetStorageSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())
	return nil
}

func newStorageManager(c cache.Cache) (*StorageManager, error) {
	lvm := lvm.New(c)
	nfs := nfs.New(c)
	m := &StorageManager{
		storages: []Storage{lvm, nfs},
	}
	return m, nil
}

func (m *StorageManager) Get(ctx *resttypes.Context) interface{} {
	cls := ctx.Object.GetID()
	fmt.Println(cls)
	for _, s := range m.storages {
		if s.GetStorageClass() == cls {
			return s.GetStroageInfo()
		}
	}
	return nil
}

func (m *StorageManager) List(ctx *resttypes.Context) interface{} {
	var infos []types.StorageInfo
	for _, s := range m.storages {
		infos = append(infos, s.GetStroageInfo())
	}
	return infos
}

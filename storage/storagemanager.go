package storage

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cluster-agent/storage/lvm"
	"github.com/zdnscloud/cluster-agent/storage/nfs"
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
	GetType() string
	GetInfo() types.Storage
}

type StorageManager struct {
	api.DefaultHandler
	storages []Storage
}

func New(c cache.Cache) (*StorageManager, error) {
	lvm, err := lvm.New(c)
	if err != nil {
		return nil, err
	}
	nfs, err := nfs.New(c)
	if err != nil {
		return nil, err
	}
	return &StorageManager{
		storages: []Storage{lvm, nfs},
	}, nil
}

func (m *StorageManager) RegisterHandler(router gin.IRoutes) error {
	schemas := resttypes.NewSchemas()
	schemas.MustImportAndCustomize(&Version, types.Storage{}, m, types.SetStorageSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())
	return nil
}

func (m *StorageManager) Get(ctx *resttypes.Context) interface{} {
	cls := ctx.Object.GetID()
	for _, s := range m.storages {
		if s.GetType() == cls {
			return s.GetInfo()
		}
	}
	return nil
}

func (m *StorageManager) List(ctx *resttypes.Context) interface{} {
	var infos []types.Storage
	for _, s := range m.storages {
		info := s.GetInfo()
		infos = append(infos, info)
	}
	return infos
}

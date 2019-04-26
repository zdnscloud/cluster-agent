package service

import (
	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "service.zcloud.cn",
	}
)

type ServiceManager struct {
	api.DefaultHandler

	cache *ServiceCache
}

func New(c cache.Cache) (*ServiceManager, error) {
	sc, err := NewServiceCache(c)
	if err != nil {
		return nil, err
	}

	return &ServiceManager{
		cache: sc,
	}, nil
}

func (m *ServiceManager) List(ctx *resttypes.Context) interface{} {
	namespace := ctx.Object.GetParent().GetID()
	switch ctx.Object.GetType() {
	case InnerServiceType:
		return m.cache.GetInnerServices(namespace)
	case OuterServiceType:
		return m.cache.GetOuterServices(namespace)
	}
	return nil
}

func (m *ServiceManager) RegisterHandler(router gin.IRoutes) error {
	schemas := resttypes.NewSchemas()

	schemas.MustImport(&Version, Namespace{})
	schemas.MustImportAndCustomize(&Version, InnerService{}, m, SetInnerServiceSchema)
	schemas.MustImportAndCustomize(&Version, OuterService{}, m, SetOuterServiceSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())
	return nil
}

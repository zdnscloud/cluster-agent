package service

import (
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
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

func (m *ServiceManager) RegisterSchemas(version *resttypes.APIVersion, schemas *resttypes.Schemas) {
	schemas.MustImport(version, Namespace{})
	schemas.MustImportAndCustomize(version, InnerService{}, m, SetInnerServiceSchema)
	schemas.MustImportAndCustomize(version, OuterService{}, m, SetOuterServiceSchema)
}

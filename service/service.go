package service

import (
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/resource"
)

type ServiceManager struct {
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

func (m *ServiceManager) List(ctx *resource.Context) interface{} {
	namespace := ctx.Resource.GetParent().GetID()
	switch ctx.Resource.GetType() {
	case resource.DefaultKindName(InnerService{}):
		return m.cache.GetInnerServices(namespace)
	case resource.DefaultKindName(OuterService{}):
		return m.cache.GetOuterServices(namespace)
	}
	return nil
}

type dumbHandler struct{}

func (h *dumbHandler) List(ctx *resource.Context) interface{} {
	return nil
}

func (m *ServiceManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, Namespace{}, &dumbHandler{})
	schemas.MustImport(version, InnerService{}, m)
	schemas.MustImport(version, OuterService{}, m)
}

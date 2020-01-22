package commonresource

import (
	"github.com/zdnscloud/gorest/resource"
)

func RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, Namespace{}, &dumbHandler{})
	schemas.MustImport(version, Deployment{}, &dumbHandler{})
	schemas.MustImport(version, DaemonSet{}, &dumbHandler{})
	schemas.MustImport(version, StatefulSet{}, &dumbHandler{})
}

type dumbHandler struct{}

func (h *dumbHandler) List(ctx *resource.Context) interface{} {
	return nil
}

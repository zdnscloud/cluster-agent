package storage

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "storage.zcloud.cn",
	}

	tokenSecret        = []byte("hello storage")
	tokenValidDuration = 24 * 3600 * time.Second
)

type App struct {
	storageManager *StorageManager
}

func NewApp() *App {
	return &App{
		storageManager: newStorageManager(),
	}
}

func (a *App) RegisterHandler(router gin.IRoutes) error {
	if err := a.registerRestHandler(router); err != nil {
		return err
	}
	return nil
}

func (a *App) registerRestHandler(router gin.IRoutes) error {
	schemas := resttypes.NewSchemas()
	schemas.MustImportAndCustomize(&Version, Storage{}, a.storageManager, SetStorageSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())
	return nil
}

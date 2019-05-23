package main

import (
	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"

	"github.com/zdnscloud/cluster-agent/network"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/cluster-agent/service"
	"github.com/zdnscloud/cluster-agent/storage"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "agent.zcloud.cn",
	}
)

func createCache() (cache.Cache, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	c, err := cache.New(config, cache.Options{})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func main() {
	log.InitLogger("debug")

	cache, err := createCache()
	if err != nil {
		log.Fatalf("Create cache failed:%s", err.Error())
	}
	stop := make(chan struct{})
	go cache.Start(stop)
	cache.WaitForCacheSync(stop)

	nodeAgentMgr := nodeagent.New()

	storageMgr, err := storage.New(cache, nodeAgentMgr)
	if err != nil {
		log.Fatalf("Create storage manager failed:%s", err.Error())
	}
	networkMgr, err := network.New(cache)
	if err != nil {
		log.Fatalf("Create network manager failed:%s", err.Error())
	}
	serviceMgr, err := service.New(cache)
	if err != nil {
		log.Fatalf("Create service manager failed:%s", err.Error())
	}

	schemas := resttypes.NewSchemas()
	storageMgr.RegisterSchemas(&Version, schemas)
	networkMgr.RegisterSchemas(&Version, schemas)
	serviceMgr.RegisterSchemas(&Version, schemas)
	nodeAgentMgr.RegisterSchemas(&Version, schemas)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		log.Fatalf("add schemas failed:%s", err.Error())
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())

	addr := "0.0.0.0:8090"
	router.Run(addr)
}

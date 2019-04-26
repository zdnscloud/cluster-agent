package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/network"
	"github.com/zdnscloud/cluster-agent/service"
	"github.com/zdnscloud/cluster-agent/storage"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client/config"
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
		panic("Create cache Error")
	}
	stop := make(chan struct{})
	go cache.Start(stop)
	cache.WaitForCacheSync(stop)

	storageMgr, err := storage.New(cache)
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

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	if err := storageMgr.RegisterHandler(router); err != nil {
		log.Fatalf("storage manager register handler failed:%s", err.Error())
	}
	if err := networkMgr.RegisterHandler(router); err != nil {
		log.Fatalf("network manager register handler failed:%s", err.Error())
	}
	if err := serviceMgr.RegisterHandler(router); err != nil {
		log.Fatalf("service manager register handler failed:%s", err.Error())
	}

	addr := "0.0.0.0:8090"
	router.Run(addr)
}

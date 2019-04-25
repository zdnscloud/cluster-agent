package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/network"
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

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	cache, err := createCache()
	if err != nil {
		panic("Create cache Error")
	}
	stop := make(chan struct{})
	go cache.Start(stop)
	cache.WaitForCacheSync(stop)

	storageMgr := storage.New(cache)
	if err := storageMgr.RegisterHandler(router); err != nil {
		log.Errorf("register storage handler failed:%s", err.Error())
		panic("Register storage handler failed")
	}
	if err := network.RegisterHandler(router); err != nil {
		log.Errorf("register network handler failed:%s", err.Error())
		panic("Register network handler failed")
	}

	addr := "0.0.0.0:8090"
	router.Run(addr)
}

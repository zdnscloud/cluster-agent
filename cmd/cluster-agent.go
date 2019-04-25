package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/network"
	"github.com/zdnscloud/cluster-agent/storage"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client/config"
)

func createCache() cache.Cache {
	config, err := config.GetConfig()
	if err != nil {
		return nil
	}

	c, err := cache.New(config, cache.Options{})
	if err != nil {
		return nil
	}
	return c
}

func main() {
	log.InitLogger(storage.LogLevel)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	cache := createCache()
	stop := make(chan struct{})
	go cache.Start(stop)
	cache.WaitForCacheSync(stop)

	if err := storage.RegisterHandler(router, cache); err != nil {
		log.Errorf("register storage handler failed:%s", err.Error())
	}
	if err := network.RegisterHandler(router); err != nil {
		log.Errorf("register network handler failed:%s", err.Error())
	}

	addr := "0.0.0.0:8090"
	router.Run(addr)
}

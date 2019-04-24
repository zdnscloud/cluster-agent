package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/network"
	"github.com/zdnscloud/cluster-agent/storage"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client/config"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
)

func createCache() cache.Cache {
	usr, err := user.Current()
	if err != nil {
		log.Errorf("Get user failed:%s", err.Error())
	}

	k8sconfig := filepath.Join(usr.HomeDir, ".kube", "config")
	f, err := os.Open(k8sconfig)
	if err != nil {
		log.Errorf("Get k8s config failed:%s", err.Error())
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Errorf("Load k8s config failed:%s", err.Error())
	}

	k8sconf, err := config.BuildConfig(data)
	if err != nil {
		log.Errorf("Build config failed:%s", err.Error())
	}

	c, err := cache.New(k8sconf, cache.Options{})
	if err != nil {
		log.Errorf("Create cache failed:%s", err.Error())
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

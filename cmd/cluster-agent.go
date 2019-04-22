package main

import (
	"github.com/gin-gonic/gin"
	//"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/network"
	"github.com/zdnscloud/cluster-agent/storage"
)

/*
func createCache() (*StorageManager, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user failed:%s", err.Error())
	}

	k8sconfig := filepath.Join(usr.HomeDir, ".kube", "config")
	f, err := os.Open(k8sconfig)
	if err != nil {
		return nil, fmt.Errorf("open %s failed:%s", k8sconfig, err.Error())
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read %s failed:%s", k8sconfig, err.Error())
	}

	k8sconf, err := config.BuildConfig(data)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster config:%s", err.Error())
	}

	stop := make(chan struct{})
	c, err := cache.New(k8sconf, cache.Options{})
	if err != nil {
		return nil, fmt.Errorf("create cache failed:%s", err.Error())
	}
	go c.Start(stop)
	c.WaitForCacheSync(stop)
}
*/
func main() {
	//log.InitLogger(Debug)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	if err := storage.RegisterHandler(router); err != nil {
		//log.Errorf("register storage handler failed:%s", err.Error())
		panic("register handler failed:" + err.Error())
	}
	if err := network.RegisterHandler(router); err != nil {
		//log.Errorf("register network handler failed:%s", err.Error())
		panic("register handler failed:" + err.Error())
	}

	addr := "0.0.0.0:8090"

	router.Run(addr)
}

package storage

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"time"
)

type StorageManager struct {
	api.DefaultHandler
	storages *StorageCache
	cache    cache.Cache
	lock     sync.RWMutex
	stopCh   chan struct{}
}

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "storage.zcloud.cn",
	}

	tokenSecret        = []byte("hello storage")
	tokenValidDuration = 24 * 3600 * time.Second
)

func RegisterHandler(router gin.IRoutes) error {
	schemas := resttypes.NewSchemas()
	m, _ := newStorageManager()
	schemas.MustImportAndCustomize(&Version, Storage{}, m, SetStorageSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())
	return nil
}

func newStorageManager() (*StorageManager, error) {
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

	ctrl := controller.New("storageCache", c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&storagev1.StorageClass{})
	ctrl.Watch(&corev1.PersistentVolume{})
	stopCh := make(chan struct{})
	m := &StorageManager{
		stopCh: stopCh,
		cache:  c,
	}
	if err := m.initStorageManagers(); err != nil {
		return nil, err
	}

	go ctrl.Start(stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	return m, nil
}

func (m *StorageManager) initStorageManagers() error {
	storages := storagev1.StorageClassList{}
	err := m.cache.List(context.TODO(), nil, &storages)
	if err != nil {
		return err
	}

	sc := newStorageCache(m.cache)
	for _, storage := range storages.Items {
		sc.OnNewStorageClass(&storage)
	}

	m.storages = sc
	return nil
}

func (m *StorageManager) List(ctx *resttypes.Context) interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.storages.GetStorages()
}

func (m *StorageManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *storagev1.StorageClass:
		m.storages.OnNewStorageClass(obj)
	case *corev1.Node:
		m.storages.OnNewNode(obj)
	case *corev1.PersistentVolume:
		m.storages.OnNewPV(obj)
	}
	return handler.Result{}, nil
}
func (m *StorageManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	return handler.Result{}, nil
}

func (m *StorageManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *storagev1.StorageClass:
		m.storages.OnDelStorageClass(obj)
	}

	return handler.Result{}, nil
}

func (m *StorageManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

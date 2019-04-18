package network

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/gorest/adaptor"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

var (
	Version = resttypes.APIVersion{
		Version: "v1",
		Group:   "network.zcloud.cn",
	}
)

func RegisterHandler(router gin.IRoutes) error {
	schemas := resttypes.NewSchemas()
	m, err := newNetworkManager()
	if err != nil {
		return fmt.Errorf("create network handler failed: %s", err.Error())
	}

	schemas.MustImportAndCustomize(&Version, Node{}, m, SetNodeSchema)
	schemas.MustImportAndCustomize(&Version, Pod{}, m, SetPodSchema)
	schemas.MustImportAndCustomize(&Version, Service{}, m, SetServiceSchema)

	server := api.NewAPIServer()
	if err := server.AddSchemas(schemas); err != nil {
		return err
	}
	server.Use(api.RestHandler)
	adaptor.RegisterHandler(router, server, server.Schemas.UrlMethods())
	return nil
}

type NetworkManager struct {
	DefaultHandler
	networks *NetworkCache
	cache    cache.Cache
	lock     sync.RWMutex
	stopCh   chan struct{}
}

func newNetworkManager() (*NetworkManager, error) {
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

	ctrl := controller.New("networkCache", c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&corev1.Pod{})
	ctrl.Watch(&corev1.Service{})

	stopCh := make(chan struct{})
	m := &NetworkManager{
		stopCh: stopCh,
		cache:  c,
	}
	if err := m.initNetworkManagers(); err != nil {
		return nil, err
	}

	go ctrl.Start(stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	return m, nil
}

func (m *NetworkManager) initNetworkManagers() error {
	nodes := &corev1.NodeList{}
	err := m.cache.List(context.TODO(), nil, nodes)
	if err != nil {
		return err
	}

	nc := newNetworkCache(m.cache)
	for _, node := range nodes.Items {
		nc.OnNewNode(&node)
	}

	m.networks = nc
	return nil
}

func (m *NetworkManager) List(ctx *resttypes.Context) interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	switch ctx.Object.GetType() {
	case NodeType:
		return m.networks.GetNodes()
	case PodType:
		return m.networks.GetPods()
	case ServiceType:
		return m.networks.GetServices()
	default:
		return nil
	}
}

func (m *NetworkManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Node:
		m.networks.OnNewNode(obj)
	case *corev1.Pod:
		m.networks.OnNewPod(obj)
	case *corev1.Service:
		m.networks.OnNewService(obj)
	}

	return handler.Result{}, nil
}

func (m *NetworkManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	switch newObj := e.ObjectNew.(type) {
	case *corev1.Service:
		m.networks.OnUpdateService(newObj)
	case *corev1.Pod:
		m.networks.OnUpdatePod(newObj)
	}

	return handler.Result{}, nil
}

func (m *NetworkManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Node:
		m.networks.OnDeleteNode(obj)
	case *corev1.Pod:
		m.networks.OnDeletePod(obj)
	case *corev1.Service:
		m.networks.OnDeleteService(obj)
	}

	return handler.Result{}, nil
}

func (m *NetworkManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

package network

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/gorest/resource"
)

type NetworkManager struct {
	networks *NetworkCache
	cache    cache.Cache
	lock     sync.RWMutex
	stopCh   chan struct{}
}

func New(c cache.Cache) (*NetworkManager, error) {
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

func (m *NetworkManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, NodeNetwork{}, m)
	schemas.MustImport(version, PodNetwork{}, m)
	schemas.MustImport(version, ServiceNetwork{}, m)
}

func (m *NetworkManager) initNetworkManagers() error {
	nodes := &corev1.NodeList{}
	err := m.cache.List(context.TODO(), nil, nodes)
	if err != nil {
		return err
	}

	nc := newNetworkCache()
	for _, node := range nodes.Items {
		nc.OnNewNode(&node)
	}

	m.networks = nc
	return nil
}

func (m *NetworkManager) List(ctx *resource.Context) interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	switch ctx.Resource.GetType() {
	case resource.DefaultKindName(NodeNetwork{}):
		return m.networks.GetNodeNetworks()
	case resource.DefaultKindName(PodNetwork{}):
		return m.networks.GetPodNetworks()
	case resource.DefaultKindName(ServiceNetwork{}):
		return m.networks.GetServiceNetworks()
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
		if e.ObjectOld.(*corev1.Service).Spec.ClusterIP != newObj.Spec.ClusterIP {
			m.networks.OnUpdateService(newObj)
		}
	case *corev1.Pod:
		m.networks.OnUpdatePod(e.ObjectOld.(*corev1.Pod), newObj)
	case *corev1.Node:
		if e.ObjectOld.(*corev1.Node).Spec.PodCIDR != newObj.Spec.PodCIDR {
			m.networks.OnUpdateNode(newObj)
		}
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

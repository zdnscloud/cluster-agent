package pvmonitor

import (
	"context"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sync"
)

type PVMonitor struct {
	DriverName string
	PVs        []types.PV
	PvAndPVC   map[string]PVC
	PVCAndPod  map[string][]types.Pod
	lock       sync.RWMutex
}

type PVC struct {
	Name           string
	NamespacedName string
}

func New(c cache.Cache, n string) (*PVMonitor, error) {
	ctrl := controller.New(n, c, scheme.Scheme)
	ctrl.Watch(&corev1.PersistentVolumeClaim{})
	ctrl.Watch(&corev1.PersistentVolume{})
	ctrl.Watch(&corev1.Pod{})
	stopCh := make(chan struct{})

	pm := &PVMonitor{
		DriverName: n,
		PVs:        make([]types.PV, 0),
		PvAndPVC:   make(map[string]PVC),
		PVCAndPod:  make(map[string][]types.Pod),
	}
	if err := pm.initPVC(c); err != nil {
		return nil, err
	}
	go ctrl.Start(stopCh, pm, predicate.NewIgnoreUnchangedUpdate())
	return pm, nil
}

func (s *PVMonitor) initPVC(c cache.Cache) error {
	pvcs := corev1.PersistentVolumeClaimList{}
	if err := c.List(context.TODO(), nil, &pvcs); err != nil {
		return err
	}
	for _, pvc := range pvcs.Items {
		s.OnNewPVC(&pvc)
	}
	return nil
}

func (s *PVMonitor) OnCreate(e event.CreateEvent) (handler.Result, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.PersistentVolume:
		s.OnNewPV(obj)
	case *corev1.PersistentVolumeClaim:
		s.OnNewPVC(obj)
	case *corev1.Pod:
		s.OnNewPod(obj)
	}
	return handler.Result{}, nil
}
func (s *PVMonitor) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	switch newObj := e.ObjectNew.(type) {
	case *corev1.PersistentVolumeClaim:
		s.OnNewPVC(newObj)
	case *corev1.PersistentVolume:
		s.OnUpdatePV(newObj)
	}
	return handler.Result{}, nil
}

func (s *PVMonitor) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.PersistentVolume:
		s.OnDelPV(obj)
	case *corev1.PersistentVolumeClaim:
		s.OnDelPVC(obj)
	case *corev1.Pod:
		s.OnDelPod(obj)
	}
	return handler.Result{}, nil
}

func (s *PVMonitor) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

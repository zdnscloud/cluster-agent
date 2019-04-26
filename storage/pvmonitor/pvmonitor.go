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
)

type PVMonitor struct {
	Name      string
	PVs       []types.PV
	PvAndPvc  map[string]PVC
	PvcAndPod map[string][]types.Pod
	PVCAndSc  map[string]string
}

type PVC struct {
	Name       string
	VolumeName string
	Pods       []types.Pod
}

func New(c cache.Cache, n string) (*PVMonitor, error) {
	ctrl := controller.New(n, c, scheme.Scheme)
	ctrl.Watch(&corev1.PersistentVolumeClaim{})
	ctrl.Watch(&corev1.PersistentVolume{})
	ctrl.Watch(&corev1.Pod{})
	stopCh := make(chan struct{})

	pm := &PVMonitor{
		Name:      n,
		PVs:       make([]types.PV, 0),
		PvAndPvc:  make(map[string]PVC),
		PvcAndPod: make(map[string][]types.Pod),
		PVCAndSc:  make(map[string]string),
	}
	if err := pm.initStorage(c); err != nil {
		return nil, err
	}
	go ctrl.Start(stopCh, pm, predicate.NewIgnoreUnchangedUpdate())
	return pm, nil
}

func (s *PVMonitor) initStorage(c cache.Cache) error {
	pods := corev1.PodList{}
	err := c.List(context.TODO(), nil, &pods)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		s.OnNewPod(&pod)
	}

	pvcs := corev1.PersistentVolumeClaimList{}
	err = c.List(context.TODO(), nil, &pvcs)
	if err != nil {
		return err
	}
	for _, pvc := range pvcs.Items {
		s.OnNewPvc(&pvc)
	}
	return nil
}

func (s *PVMonitor) OnCreate(e event.CreateEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.PersistentVolume:
		s.OnNewPV(obj)
	case *corev1.PersistentVolumeClaim:
		s.OnNewPvc(obj)
	case *corev1.Pod:
		s.OnNewPod(obj)
	}
	return handler.Result{}, nil
}
func (s *PVMonitor) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch newObj := e.ObjectNew.(type) {
	case *corev1.PersistentVolumeClaim:
		s.OnNewPvc(newObj)
	}
	return handler.Result{}, nil
}

func (s *PVMonitor) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.PersistentVolume:
		s.OnDelPV(obj)
	case *corev1.PersistentVolumeClaim:
		s.OnDelPvc(obj)
	case *corev1.Pod:
		s.OnDelPod(obj)
	}
	return handler.Result{}, nil
}

func (s *PVMonitor) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

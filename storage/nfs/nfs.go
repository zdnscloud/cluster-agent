package nfs

import (
	"context"
	//"fmt"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func New(c cache.Cache) *Storage {
	ctrl := controller.New(CtrlName, c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&storagev1.StorageClass{})
	ctrl.Watch(&corev1.PersistentVolume{})
	ctrl.Watch(&corev1.Pod{})
	ctrl.Watch(&corev1.PersistentVolumeClaim{})

	stopCh := make(chan struct{})

	//nm := lvmd.NewNodeManager(c, "k8s")
	//a := nm.GetNodes()
	//fmt.Println(a)

	res := &Storage{
		Name:      "nfs",
		PvAndPvc:  make(map[string]types.Pvc),
		PvcAndPod: make(map[string][]types.Pod),
	}
	if err := res.initLvm(c); err != nil {
		return nil
	}
	go ctrl.Start(stopCh, res, predicate.NewIgnoreUnchangedUpdate())
	return res
}

func (s *Storage) initLvm(c cache.Cache) error {
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

func (s *Storage) GetStorageClass() string {
	return "nfs"
}

func (s *Storage) GetStroageInfo() types.StorageInfo {
	//fmt.Println(s.PvcAndPod)
	pvs := s.PVs
	var res []types.PV
	for _, p := range pvs {
		pvc := s.PvAndPvc[p.Name].Name
		pods := s.PvcAndPod[pvc]
		pv := types.PV{
			Name: p.Name,
			Size: p.Size,
			Pods: pods,
		}
		res = append(res, pv)
	}
	tmp := &types.StorageInfo{
		Name: "nfs",
		PVs:  res,
	}
	return *tmp
}

func (s *Storage) OnCreate(e event.CreateEvent) (handler.Result, error) {
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
func (s *Storage) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	/*
		switch newObj := e.ObjectNew.(type) {
		case *corev1.PersistentVolumeClaim:
			s.OnUpdatePvc(newObj)
		}*/
	return handler.Result{}, nil
}

func (s *Storage) OnDelete(e event.DeleteEvent) (handler.Result, error) {
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

func (s *Storage) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

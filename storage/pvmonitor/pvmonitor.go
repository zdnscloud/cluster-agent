package pvmonitor

import (
	"sort"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
)

type PVMonitor struct {
	Pvs  map[string]*types.PV
	Pvcs map[string]string
	Pods map[string][]types.Pod
	lock sync.RWMutex
}

func New(c cache.Cache) (*PVMonitor, error) {
	ctrl := controller.New("volume", c, scheme.Scheme)
	ctrl.Watch(&corev1.PersistentVolumeClaim{})
	ctrl.Watch(&corev1.PersistentVolume{})
	ctrl.Watch(&corev1.Pod{})
	stopCh := make(chan struct{})

	pm := &PVMonitor{
		Pvs:  make(map[string]*types.PV),
		Pvcs: make(map[string]string),
		Pods: make(map[string][]types.Pod),
	}
	go ctrl.Start(stopCh, pm, predicate.NewIgnoreUnchangedUpdate())
	return pm, nil
}

func (s *PVMonitor) OnCreate(e event.CreateEvent) (handler.Result, error) {
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
	switch newObj := e.ObjectNew.(type) {
	case *corev1.PersistentVolumeClaim:
		s.OnNewPVC(newObj)
	case *corev1.PersistentVolume:
		s.OnUpdatePV(newObj)
	}
	return handler.Result{}, nil
}

func (s *PVMonitor) OnDelete(e event.DeleteEvent) (handler.Result, error) {
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

func (s *PVMonitor) Classify(mountpoints map[string][]int64) map[string]types.Pvs {
	s.lock.Lock()
	defer s.lock.Unlock()
	res := make(map[string]types.Pvs)
	for _, pv := range s.Pvs {
		if _, ok := res[pv.StorageClassName]; !ok {
			res[pv.StorageClassName] = types.Pvs{}
		}
		usedSize, freeSize := utils.GetPVSize(pv, mountpoints)
		pvc := s.Pvcs[pv.Name]
		res[pv.StorageClassName] = append(res[pv.StorageClassName], &types.PV{
			Name:     pv.Name,
			Size:     pv.Size,
			UsedSize: usedSize,
			FreeSize: freeSize,
			Pods:     s.Pods[pvc],
			Node:     pv.Node,
			PVC:      pvc,
		})
		sort.Sort(res[pv.StorageClassName])
	}
	return res
}

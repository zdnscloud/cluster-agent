package storageclass

import (
	"context"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	"github.com/zdnscloud/lvmd.git"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func New(c cache.Cache, n string) (*StorageCache, error) {
	ctrl := controller.New(CtrlName, c, scheme.Scheme)
	ctrl.Watch(&corev1.Node{})
	ctrl.Watch(&storagev1.StorageClass{})
	ctrl.Watch(&corev1.PersistentVolume{})
	ctrl.Watch(&corev1.Pod{})
	ctrl.Watch(&corev1.PersistentVolumeClaim{})
	stopCh := make(chan struct{})

	var nodes []types.Node
	if n == "lvm" {
		nodes = getNodes(c)
	}
	res := &StorageCache{
		Name:      n,
		Nodes:     nodes,
		PvAndPvc:  make(map[string]types.Pvc),
		PvcAndPod: make(map[string][]types.Pod),
	}
	if err := res.initStorage(c); err != nil {
		return nil, err
	}
	go ctrl.Start(stopCh, res, predicate.NewIgnoreUnchangedUpdate())
	return res, nil
}

func getNodes(c cache.Cache) []types.Node {
	nm := lvmd.NewNodeManager(c, CSIDefaultVgName)
	ns := nm.GetNodes()
	var nodes []types.Node
	for _, v := range ns {
		node := types.Node{
			Name:     v.Name,
			Size:     byteToGb(v.Size),
			FreeSize: byteToGb(v.FreeSize),
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func (s *StorageCache) initStorage(c cache.Cache) error {
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

func (s *StorageCache) GetStorageClass() string {
	return s.Name
}

func (s *StorageCache) GetStroageInfo(cls string) types.StorageInfo {
	pvs := s.PVs
	var res []types.PV
	for _, p := range pvs {
		pvc := s.PvAndPvc[p.Name].Name
		if p.StorageClassName == cls {
			pods := s.PvcAndPod[pvc]
			pv := types.PV{
				Name: p.Name,
				Size: p.Size,
				Pods: pods,
			}
			res = append(res, pv)
		}
	}
	var tsize, fsize int
	switch cls {
	case "lvm":
		tsize, fsize = s.getLVMSize(s.Nodes)
	case "nfs":
		tsize, fsize = s.getNFSSize(ZKENFSPvcName)
	}

	tmp := &types.StorageInfo{
		Name:     s.Name,
		Size:     tsize,
		FreeSize: fsize,
		Nodes:    s.Nodes,
		PVs:      res,
	}
	return *tmp
}

func (s *StorageCache) getLVMSize(nodes []types.Node) (int, int) {
	var tsize, fsize int
	for _, n := range nodes {
		tsize += n.Size
		fsize += n.FreeSize
	}
	return tsize, fsize
}

func (s *StorageCache) getNFSSize(n string) (int, int) {
	var pv types.PV
	for k, v := range s.PvAndPvc {
		if v.Name == n {
			for i, p := range s.PVs {
				if p.Name == k {
					pv = s.PVs[i]
				}
			}
			break
		}
	}
	return pv.Size, pv.Size
}

func (s *StorageCache) OnCreate(e event.CreateEvent) (handler.Result, error) {
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
func (s *StorageCache) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch newObj := e.ObjectNew.(type) {
	case *corev1.PersistentVolumeClaim:
		s.OnNewPvc(newObj)
	}
	return handler.Result{}, nil
}

func (s *StorageCache) OnDelete(e event.DeleteEvent) (handler.Result, error) {
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

func (s *StorageCache) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

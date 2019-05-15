package nfs

import (
	"context"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	corev1 "k8s.io/api/core/v1"
)

const (
	NFSStorageClassName = "nfs"
	ZKEStorageNamespace = "kube-storage"
	ZKENFSPvcName       = "nfs-data-nfs-provisioner-0"
)

type NFS struct {
	Size     string
	FreeSize string
	UsedSize string
	PVData   *pvmonitor.PVMonitor
	Cache    cache.Cache
}

func New(c cache.Cache) (*NFS, error) {
	pm, err := pvmonitor.New(c, NFSStorageClassName)
	if err != nil {
		return nil, err
	}
	nfs := &NFS{
		PVData: pm,
		Cache:  c,
	}
	nfs.setNFSSize()
	return nfs, nil
}

func (s *NFS) GetType() string {
	return NFSStorageClassName
}

func (s *NFS) GetInfo() types.Storage {
	pvs := s.PVData.PVs
	var res []types.PV
	for _, p := range pvs {
		pvc := s.PVData.PvAndPVC[p.Name].Name
		pods := s.PVData.PVCAndPod[pvc]
		pv := types.PV{
			Name: p.Name,
			Size: p.Size,
			Pods: pods,
		}
		res = append(res, pv)
	}

	return types.Storage{
		Name:     NFSStorageClassName,
		Size:     s.Size,
		FreeSize: s.FreeSize,
		UsedSize: s.UsedSize,
		PVs:      res,
	}
}

func (s *NFS) setNFSSize() {
	var pvsize string
	pvcs := corev1.PersistentVolumeClaimList{}
	err := s.Cache.List(context.TODO(), nil, &pvcs)
	if err != nil {
		log.Errorf("List pvc failed:%s", err.Error())
	}
	for _, pvc := range pvcs.Items {
		if pvc.Namespace == ZKEStorageNamespace && pvc.Name == ZKENFSPvcName {
			quantity := pvc.Spec.Resources.Requests["storage"]
			pvsize = utils.SizetoGb(quantity)
		}
	}
	s.Size = pvsize
	s.FreeSize = "0.00"
	s.UsedSize = pvsize
}

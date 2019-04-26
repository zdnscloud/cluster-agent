package nfs

import (
	"context"
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	corev1 "k8s.io/api/core/v1"
)

func New(c cache.Cache) (*NFS, error) {
	pm, err := pvmonitor.New(c, SourceName)
	if err != nil {
		return nil, err
	}
	tsize, fsize := getNFSSize(c)
	res := &NFS{
		Name:     SourceName,
		Size:     tsize,
		FreeSize: fsize,
		PVData:   pm,
		Cache:    c,
	}
	return res, nil
}

func (s *NFS) GetType() string {
	return s.Name
}

func (s *NFS) GetInfo() types.Storage {
	pvs := s.PVData.PVs
	var res []types.PV
	for _, p := range pvs {
		pvc := s.PVData.PvAndPvc[p.Name].Name
		pods := s.PVData.PvcAndPod[pvc]
		pv := types.PV{
			Name: p.Name,
			Size: p.Size,
			Pods: pods,
		}
		res = append(res, pv)
	}

	tmp := &types.Storage{
		Name:     s.Name,
		Size:     s.Size,
		FreeSize: s.FreeSize,
		PVs:      res,
	}
	return *tmp
}

func getNFSSize(c cache.Cache) (int, int) {
	pvcs := corev1.PersistentVolumeClaimList{}
	err := c.List(context.TODO(), nil, &pvcs)
	if err != nil {
		return 0, 0
	}
	for _, pvc := range pvcs.Items {
		if pvc.Namespace == ZKEStorageNamespace && pvc.Name == ZKENFSPvcName {
			quantity := pvc.Spec.Resources.Requests["storage"]
			pvsize := utils.Sizetog(quantity)
			return pvsize, pvsize
		}
	}
	return 0, 0
}

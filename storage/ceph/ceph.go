package ceph

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"sync"
)

const (
	CephStorageType       = "cephfs"
	CephStorageDriverName = "cephfs.csi.ceph.com"
)

type ceph struct {
	PVData *pvmonitor.PVMonitor
	Cache  cache.Cache
	lock   sync.RWMutex
}

func New(c cache.Cache) (*ceph, error) {
	pm, err := pvmonitor.New(c, CephStorageDriverName)
	if err != nil {
		return nil, err
	}
	return &ceph{
		PVData: pm,
		Cache:  c,
	}, nil
}

func (s *ceph) GetType() string {
	return CephStorageType
}

func (s *ceph) GetInfo(mountpoints map[string][]int64) *types.Storage {
	s.lock.Lock()
	defer s.lock.Unlock()
	var res []types.PV
	for _, p := range s.PVData.PVs {
		uSize, fSize := utils.GetPVSize(p, mountpoints)
		pvc := s.PVData.PvAndPVC[p.Name].Name
		pods := s.PVData.PVCAndPod[pvc]
		pv := types.PV{
			Name:     p.Name,
			Size:     p.Size,
			UsedSize: uSize,
			FreeSize: fSize,
			Pods:     pods,
			PVC:      pvc,
		}
		res = append(res, pv)
	}
	return &types.Storage{
		Name: CephStorageType,
		PVs:  res,
	}
}

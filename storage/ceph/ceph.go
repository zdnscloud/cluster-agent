package ceph

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"sync"
)

const (
	CephStorageType      = "ceph"
	CephStorageClassName = "cephfs"
)

type ceph struct {
	Nodes    []types.Node
	Size     string
	FreeSize string
	UsedSize string
	PVData   *pvmonitor.PVMonitor
	Cache    cache.Cache
	lock     sync.RWMutex
}

func New(c cache.Cache) (*ceph, error) {
	pm, err := pvmonitor.New(c, CephStorageClassName)
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
	pvs := s.PVData.PVs
	var res []types.PV
	for _, p := range pvs {
		uSize, fSize := utils.GetPVSize(p, mountpoints)
		pvc := s.PVData.PvAndPVC[p.Name].Name
		pods := s.PVData.PVCAndPod[pvc]
		pv := types.PV{
			Name:     p.Name,
			Size:     p.Size,
			UsedSize: uSize,
			FreeSize: fSize,
			Pods:     pods,
		}
		res = append(res, pv)
	}
	nodes, tSize, uSize, fSize, err := utils.GetNodesCapacity(CephStorageType)
	if err != nil {
		_ = err
	}
	return &types.Storage{
		Name:     CephStorageType,
		Size:     tSize,
		UsedSize: uSize,
		FreeSize: fSize,
		Nodes:    nodes,
		PVs:      res,
	}
}

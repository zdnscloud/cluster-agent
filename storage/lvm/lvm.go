package lvm

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"sync"
)

const (
	LvmStorageType      = "lvm"
	LvmStorageClassName = "lvm"
)

type LVM struct {
	Nodes    []types.Node
	Size     string
	FreeSize string
	UsedSize string
	PVData   *pvmonitor.PVMonitor
	Cache    cache.Cache
	lock     sync.RWMutex
}

func New(c cache.Cache) (*LVM, error) {
	pm, err := pvmonitor.New(c, LvmStorageClassName)
	if err != nil {
		return nil, err
	}
	return &LVM{
		PVData: pm,
		Cache:  c,
	}, nil
}

func (s *LVM) GetType() string {
	return LvmStorageType
}

func (s *LVM) GetInfo(mountpoints map[string][]int64) *types.Storage {
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
	nodes, tSize, uSize, fSize, err := utils.GetNodesCapacity(LvmStorageType)
	if err != nil {
		_ = err
	}
	return &types.Storage{
		Name:     LvmStorageType,
		Size:     tSize,
		UsedSize: uSize,
		FreeSize: fSize,
		Nodes:    nodes,
		PVs:      res,
	}
}

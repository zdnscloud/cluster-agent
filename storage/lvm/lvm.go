package lvm

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"sync"
)

const (
	LvmStorageType       = "lvm"
	LvmStorageDriverName = "csi-lvmplugin"
)

type LVM struct {
	PVData *pvmonitor.PVMonitor
	Cache  cache.Cache
	lock   sync.RWMutex
}

func New(c cache.Cache) (*LVM, error) {
	pm, err := pvmonitor.New(c, LvmStorageDriverName)
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
	var res []types.PV
	for _, p := range s.PVData.PVs {
		uSize, fSize := utils.GetPVSize(p, mountpoints)
		pvc := s.PVData.PvAndPVC[p.Name].Name
		pods := s.PVData.PVCAndPod[pvc]
		node, _ := utils.GetNodeForLvmPv(p.Name, LvmStorageDriverName)
		pv := types.PV{
			Name:     p.Name,
			Size:     p.Size,
			UsedSize: uSize,
			FreeSize: fSize,
			Pods:     pods,
			Node:     node,
			PVC:      pvc,
		}
		res = append(res, pv)
	}
	return &types.Storage{
		Name: LvmStorageType,
		PVs:  res,
	}
}

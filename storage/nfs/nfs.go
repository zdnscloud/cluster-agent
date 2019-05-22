package nfs

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"strings"
	"sync"
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
	lock     sync.RWMutex
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
	return nfs, nil
}

func (s *NFS) GetType() string {
	return NFSStorageClassName
}

func (s *NFS) GetInfo(mountpoints map[string]int64) types.Storage {
	s.lock.Lock()
	defer s.lock.Unlock()
	pvs := s.PVData.PVs
	var res []types.PV
	for _, p := range pvs {
		var uSize, fSize string
		for k, v := range mountpoints {
			if strings.Contains(k, p.Name) {
				uSize = utils.ByteToGbiTos(v)
				fSize = utils.GetFree(p.Size, v)
			}
		}
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
	tSize, uSize, fSize, _ := utils.GetNFSSize()
	return types.Storage{
		Name:     NFSStorageClassName,
		Size:     tSize,
		FreeSize: uSize,
		UsedSize: fSize,
		PVs:      res,
	}
}

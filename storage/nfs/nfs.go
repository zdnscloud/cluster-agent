package nfs

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"sync"
)

const (
	NFSStorageClassName = "nfs"
)

type NFS struct {
	Nodes    []types.Node
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

func (s *NFS) GetInfo(mountpoints map[string][]int64) types.Storage {
	s.lock.Lock()
	defer s.lock.Unlock()
	pvs := s.PVData.PVs
	var res []types.PV
	for _, p := range pvs {
		uSize, fSize := utils.GetNFSPVSize(p, mountpoints)
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
	var tSize, uSize, fSize string
	v, ok := mountpoints[utils.NFSHostMonitPath]
	if ok {
		tSize = utils.KbyteToGb(v[0])
		uSize = utils.KbyteToGb(v[1])
		fSize = utils.CountFreeSize(tSize, v[1])
	}
	s.setNodes(tSize, uSize, fSize)
	return types.Storage{
		Name:     NFSStorageClassName,
		Size:     tSize,
		FreeSize: uSize,
		UsedSize: fSize,
		PVs:      res,
		Nodes:    s.Nodes,
	}
}

func (s *NFS) setNodes(t string, u string, f string) {
	var nodes []types.Node
	ns, err := utils.GetNodes()
	if err != nil {
		log.Warnf("Set NFS Nodes failed:%s", err.Error())
		return
	}
	for _, n := range ns.Items {
		if n.Labels[utils.StorageHostLabels] != utils.StorageNFSHostLabelsValue {
			continue
		}
		node := types.Node{
			Name:     n.Name,
			Size:     t,
			UsedSize: u,
			FreeSize: f,
		}
		nodes = append(nodes, node)
	}
	s.Nodes = nodes
}

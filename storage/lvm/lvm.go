package lvm

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/lvmd"
	"strconv"
	"sync"
)

const (
	LVMStorageClassName = "lvm"
	CSIDefaultVgName    = "k8s"
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
	pm, err := pvmonitor.New(c, LVMStorageClassName)
	if err != nil {
		return nil, err
	}
	lvm := &LVM{
		PVData: pm,
		Cache:  c,
	}

	return lvm, nil
}

func (s *LVM) GetType() string {
	return LVMStorageClassName
}

func (s *LVM) GetInfo(mountpoints map[string][]int64) types.Storage {
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
	s.SetNodes()
	s.SetSize()

	return types.Storage{
		Name:     LVMStorageClassName,
		Size:     s.Size,
		UsedSize: s.UsedSize,
		FreeSize: s.FreeSize,
		Nodes:    s.Nodes,
		PVs:      res,
	}
}

func (s *LVM) SetNodes() {
	nm := lvmd.NewNodeManager(s.Cache, CSIDefaultVgName)
	ns := nm.GetNodes()
	var nodes []types.Node
	for _, v := range ns {
		node := types.Node{
			Name:     v.Name,
			Size:     utils.ByteToGb(v.Size),
			UsedSize: utils.ByteToGb((v.Size - v.FreeSize)),
			FreeSize: utils.ByteToGb(v.FreeSize),
		}
		nodes = append(nodes, node)
	}
	s.Nodes = nodes
}

func (s *LVM) SetSize() {
	var tsize, fsize, usize float64
	for _, n := range s.Nodes {
		t, _ := strconv.ParseFloat(n.Size, 64)
		f, _ := strconv.ParseFloat(n.FreeSize, 64)
		u, _ := strconv.ParseFloat(n.UsedSize, 64)
		tsize += t
		fsize += f
		usize += u
	}
	s.Size = strconv.FormatFloat(tsize, 'f', -1, 64)
	s.FreeSize = strconv.FormatFloat(fsize, 'f', -1, 64)
	s.UsedSize = strconv.FormatFloat(usize, 'f', -1, 64)
}

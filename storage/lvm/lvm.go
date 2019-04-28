package lvm

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/lvmd.git"
)

const (
	LVMStorageClassName = "lvm"
	CSIDefaultVgName    = "k8s"
)

type LVM struct {
	Nodes    []types.Node
	Size     int
	FreeSize int
	PVData   *pvmonitor.PVMonitor
	Cache    cache.Cache
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

func (s *LVM) GetInfo() types.Storage {
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
	s.SetNodes()
	s.SetSize()

	return types.Storage{
		Name:     LVMStorageClassName,
		Size:     s.Size,
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
			FreeSize: utils.ByteToGb(v.FreeSize),
		}
		nodes = append(nodes, node)
	}
	s.Nodes = nodes
}

func (s *LVM) SetSize() {
	var tsize, fsize int
	for _, n := range s.Nodes {
		tsize += n.Size
		fsize += n.FreeSize
	}
	s.Size = tsize
	s.FreeSize = fsize
}

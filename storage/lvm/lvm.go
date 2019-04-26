package lvm

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/lvmd.git"
)

func New(c cache.Cache) (*LVM, error) {
	pm, err := pvmonitor.New(c, SourceName)
	if err != nil {
		return nil, err
	}
	nodes := getNodes(c)
	tsize, fsize := getLVMSize(nodes)
	res := &LVM{
		Name:     SourceName,
		Nodes:    nodes,
		Size:     tsize,
		FreeSize: fsize,
		PVData:   pm,
		Cache:    c,
	}
	return res, nil
}

func (s *LVM) GetType() string {
	return s.Name
}

func (s *LVM) GetInfo() types.Storage {
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
		Nodes:    s.Nodes,
		PVs:      res,
	}
	return *tmp
}

func getNodes(c cache.Cache) []types.Node {
	nm := lvmd.NewNodeManager(c, CSIDefaultVgName)
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
	return nodes
}

func getLVMSize(nodes []types.Node) (int, int) {
	var tsize, fsize int
	for _, n := range nodes {
		tsize += n.Size
		fsize += n.FreeSize
	}
	return tsize, fsize
}

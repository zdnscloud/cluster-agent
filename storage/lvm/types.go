package lvm

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/cache"
)

const (
	SourceName       = "lvm"
	CSIDefaultVgName = "k8s"
)

type LVM struct {
	Name     string
	Nodes    []types.Node
	Size     int
	FreeSize int
	PVData   *pvmonitor.PVMonitor
	Cache    cache.Cache
}

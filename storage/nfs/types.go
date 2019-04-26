package nfs

import (
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/gok8s/cache"
)

const (
	SourceName          = "nfs"
	ZKEStorageNamespace = "zcloud"
	ZKENFSPvcName       = "nfs-data-nfs-provisioner-0"
)

type NFS struct {
	Name     string
	Size     int
	FreeSize int
	PVData   *pvmonitor.PVMonitor
	Cache    cache.Cache
}

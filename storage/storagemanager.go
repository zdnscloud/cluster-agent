package storage

import (
	cementcache "github.com/zdnscloud/cement/cache"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/cluster-agent/storage/ceph"
	"github.com/zdnscloud/cluster-agent/storage/lvm"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"time"
)

type Storage interface {
	GetType() string
	GetInfo(map[string][]int64) *types.Storage
}

type StorageManager struct {
	api.DefaultHandler

	storages     []Storage
	NodeAgentMgr *nodeagent.NodeAgentManager
	cache        *cementcache.Cache
	timeout      int
}

func New(c cache.Cache, to int, nodeAgentMgr *nodeagent.NodeAgentManager) (*StorageManager, error) {
	lvm, err := lvm.New(c)
	if err != nil {
		return nil, err
	}
	ceph, err := ceph.New(c)
	if err != nil {
		return nil, err
	}
	return &StorageManager{
		storages:     []Storage{lvm, ceph},
		NodeAgentMgr: nodeAgentMgr,
		cache:        cementcache.New(1, hashMountPoints, false),
		timeout:      to,
	}, nil
}

func (m *StorageManager) RegisterSchemas(version *resttypes.APIVersion, schemas *resttypes.Schemas) {
	schemas.MustImportAndCustomize(version, types.Storage{}, m, types.SetStorageSchema)
}

func (m *StorageManager) Get(ctx *resttypes.Context) interface{} {
	cls := ctx.Object.GetID()
	mountpoints := m.GetBuf()
	if len(mountpoints) == 0 {
		log.Infof("Get pv used info from nodeagent")
		log.Infof("Add cache %d second", m.timeout)
		mountpoints = m.SetBuf()
	}
	for _, s := range m.storages {
		if s.GetType() == cls {
			return s.GetInfo(mountpoints)
		}
	}
	return nil
}

func (m *StorageManager) List(ctx *resttypes.Context) interface{} {
	var infos []*types.Storage
	mountpoints := m.GetBuf()
	if len(mountpoints) == 0 {
		log.Infof("Get pv used info from nodeagent")
		log.Infof("Add cache %d second", m.timeout)
		mountpoints = m.SetBuf()
	}
	for _, s := range m.storages {
		infos = append(infos, s.GetInfo(mountpoints))
	}
	return infos
}

var key = cementcache.HashString("1")

func hashMountPoints(s cementcache.Value) cementcache.Key {
	return key
}

func (m *StorageManager) SetBuf() map[string][]int64 {
	mountpoints, err := utils.GetAllPvUsedSize(m.NodeAgentMgr)
	if err != nil {
		log.Warnf("Get PV Used Size failed:%s", err.Error())
	}
	if len(mountpoints) == 0 {
		log.Warnf("Has no info to cache")
		return mountpoints
	}
	m.cache.Add(&mountpoints, time.Duration(m.timeout)*time.Second)
	return mountpoints
}

func (m *StorageManager) GetBuf() map[string][]int64 {
	log.Infof("Get pv used info from cache")
	mountpoints := make(map[string][]int64)
	res, has := m.cache.Get(key)
	if !has {
		log.Warnf("Cache not found info")
		return mountpoints
	}
	mountpoints = *res.(*map[string][]int64)
	return mountpoints
}

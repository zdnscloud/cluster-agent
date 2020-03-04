package storage

import (
	cementcache "github.com/zdnscloud/cement/cache"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/cluster-agent/storage/pvmonitor"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/resource"
	"time"
)

type StorageManager struct {
	pvmonitor    *pvmonitor.PVMonitor
	NodeAgentMgr *nodeagent.NodeAgentManager
	cache        *cementcache.Cache
	timeout      int
}

func New(c cache.Cache, to int, nodeAgentMgr *nodeagent.NodeAgentManager) (*StorageManager, error) {
	pm, err := pvmonitor.New(c)
	if err != nil {
		return nil, err
	}
	return &StorageManager{
		pvmonitor:    pm,
		NodeAgentMgr: nodeAgentMgr,
		cache:        cementcache.New(1, hashMountPoints, false),
		timeout:      to,
	}, nil
}

func (m *StorageManager) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, types.Storage{}, m)
}

func (m *StorageManager) Get(ctx *resource.Context) resource.Resource {
	res := ctx.Resource.(*types.Storage)
	sc := ctx.Resource.GetID()
	mountpoints := m.GetBuf()
	if len(mountpoints) == 0 {
		mountpoints = m.SetBuf()
	}
	infos := m.pvmonitor.Classify(mountpoints)
	if pvs, ok := infos[sc]; ok {
		res.Name = sc
		res.PVs = pvs
	}
	res.SetID(sc)
	return res
}

func (m *StorageManager) List(ctx *resource.Context) interface{} {
	var infos []*types.Storage
	mountpoints := m.GetBuf()
	if len(mountpoints) == 0 {
		mountpoints = m.SetBuf()
	}
	for c, pvs := range m.pvmonitor.Classify(mountpoints) {
		infos = append(infos, &types.Storage{
			Name: c,
			PVs:  pvs,
		})
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
		return mountpoints
	}
	m.cache.Add(&mountpoints, time.Duration(m.timeout)*time.Second)
	return mountpoints
}

func (m *StorageManager) GetBuf() map[string][]int64 {
	mountpoints := make(map[string][]int64)
	res, has := m.cache.Get(key)
	if !has {
		return mountpoints
	}
	mountpoints = *res.(*map[string][]int64)
	return mountpoints
}

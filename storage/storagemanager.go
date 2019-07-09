package storage

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/cluster-agent/storage/ceph"
	"github.com/zdnscloud/cluster-agent/storage/lvm"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

type Storage interface {
	GetType() string
	GetInfo(map[string][]int64) *types.Storage
}

type StorageManager struct {
	api.DefaultHandler

	storages     []Storage
	NodeAgentMgr *nodeagent.NodeAgentManager
}

func New(c cache.Cache, nodeAgentMgr *nodeagent.NodeAgentManager) (*StorageManager, error) {
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
	}, nil
}

func (m *StorageManager) RegisterSchemas(version *resttypes.APIVersion, schemas *resttypes.Schemas) {
	schemas.MustImportAndCustomize(version, types.Storage{}, m, types.SetStorageSchema)
}

func (m *StorageManager) Get(ctx *resttypes.Context) interface{} {
	cls := ctx.Object.GetID()
	mountpoints, err := utils.GetAllPvUsedSize(m.NodeAgentMgr)
	if err != nil {
		log.Warnf("Get PV Used Size failed:%s", err.Error())
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
	mountpoints, err := utils.GetAllPvUsedSize(m.NodeAgentMgr)
	if err != nil {
		log.Warnf("Get PV Used Size failed:%s", err.Error())
	}
	for _, s := range m.storages {
		infos = append(infos, s.GetInfo(mountpoints))
	}
	return infos
}

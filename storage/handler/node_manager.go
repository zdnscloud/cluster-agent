package handler

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

type NodeManager struct {
	DefaultHandler
	storages *StorageManager
}

func newNodeManager(storages *StorageManager) *NodeManager {
	return &NodeManager{storages: storages}
}

func (m *NodeManager) List(ctx *resttypes.Context) interface{} {
	return getNodeSlice()
}

func (m *NodeManager) Get(ctx *resttypes.Context) interface{} {
	id := ctx.Object.GetID()
	nodestmp := getNodeSlice()
	var nodes []Node
	for _, v := range nodestmp {
		if id == v.Name {
			nodes = append(nodes, v)
		}
	}
	return nodes
}

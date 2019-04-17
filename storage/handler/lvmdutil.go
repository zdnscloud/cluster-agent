package handler

import (
	"context"
	lvmd "github.com/google/lvmd/proto"
	"github.com/zdnscloud/cluster-agent/storage/lvmdclient"
	"time"
)

func GetVG(node string) ([]*lvmd.VolumeGroup, error) {
	addr := node + ":" + LvmdPort
	tmout := time.Second * ConTimeout
	conn, err := lvmdclient.NewLVMConnection(addr, tmout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx := context.TODO()
	resp, err := conn.GetVG(ctx)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

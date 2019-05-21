package utils

import (
	"context"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	nodeclient "github.com/zdnscloud/node-agent/client"
	pb "github.com/zdnscloud/node-agent/proto"
	corev1 "k8s.io/api/core/v1"
	"time"
)

const (
	ZkeInternalIPAnnKey = "zdnscloud.cn/internal-ip"
)

func getNodes() ([]string, error) {
	config, err := config.GetConfig()
	cli, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}
	nodes := corev1.NodeList{}
	err = cli.List(context.TODO(), nil, &nodes)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0)
	for _, n := range nodes.Items {
		addrs = append(addrs, n.Annotations[ZkeInternalIPAnnKey])
	}
	return addrs, nil
}

func GetAllPvUsedSize() map[string]int64 {
	infos := make(map[string]int64)
	addrs, err := getNodes()
	if err != nil {
		return infos
	}
	for _, ip := range addrs {
		addr := ip + ":8899"
		cli, err := nodeclient.NewClient(addr, 10*time.Second)
		if err != nil {
			return infos
		}
		req := pb.GetBlockUsedSizeRequest{}
		reply, err := cli.GetBlockUsedSizeSize(context.TODO(), &req)
		if err != nil {
			return infos
		}
		for k, v := range reply.Infos {
			infos[k] = v
		}
	}
	return infos
}

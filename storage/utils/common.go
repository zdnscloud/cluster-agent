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
	StorageHostLabels   = "storage.zcloud.cn/storagetype"
	NFSHostMonitPath    = "/var/lib/singlecloud/nfs-export"
)

func getNodes() (corev1.NodeList, error) {
	nodes := corev1.NodeList{}
	config, err := config.GetConfig()
	cli, err := client.New(config, client.Options{})
	if err != nil {
		return nodes, err
	}
	err = cli.List(context.TODO(), nil, &nodes)
	if err != nil {
		return nodes, err
	}
	return nodes, nil
}

func GetAllPvUsedSize() (map[string]int64, error) {
	infos := make(map[string]int64)
	nodes, err := getNodes()
	if err != nil {
		return infos, err
	}
	for _, n := range nodes.Items {
		addr := n.Annotations[ZkeInternalIPAnnKey] + ":8899"
		cli, err := nodeclient.NewClient(addr, 10*time.Second)
		if err != nil {
			continue
		}
		req := pb.GetBlockUsedSizeRequest{}
		reply, err := cli.GetBlockUsedSizeSize(context.TODO(), &req)
		if err != nil {
			continue
		}
		for k, v := range reply.Infos {
			infos[k] = v
		}
	}
	return infos, err
}

func GetNFSSize() (string, string, string, error) {
	var totalsize, usedsize, freesize string
	nodes, err := getNodes()
	if err != nil {
		return totalsize, usedsize, freesize, err
	}
	for _, n := range nodes.Items {
		if n.Labels[StorageHostLabels] == "Nfs" {
			addr := n.Annotations[ZkeInternalIPAnnKey] + ":8899"
			cli, err := nodeclient.NewClient(addr, 10*time.Second)
			if err != nil {
				return totalsize, usedsize, freesize, err
			}
			tSize, err := getNFSSize(cli, "t")
			if err != nil {
				return totalsize, usedsize, freesize, err
			}
			totalsize = ByteToGbiTos(tSize)
			uSize, err := getNFSSize(cli, "u")
			if err != nil {
				return totalsize, usedsize, freesize, err
			}
			usedsize = ByteToGbiTos(uSize)
			fSize, err := getNFSSize(cli, "f")
			if err != nil {
				return totalsize, usedsize, freesize, err
			}
			freesize = ByteToGbiTos(fSize)
		}
	}
	return totalsize, usedsize, freesize, nil
}

func getNFSSize(cli pb.NodeAgentClient, t string) (int64, error) {
	req := pb.GetBlockUsedSizeRequest{
		//	Path: NFSHostMonitPath,
		//	Type: t,
	}
	reply, err := cli.GetBlockUsedSizeSize(context.TODO(), &req)
	if err != nil {
		return int64(0), err
	}
	return reply.Infos[NFSHostMonitPath], nil
}

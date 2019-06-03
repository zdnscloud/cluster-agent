package utils

import (
	"context"
	"fmt"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	nodeclient "github.com/zdnscloud/node-agent/client"
	pb "github.com/zdnscloud/node-agent/proto"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	StorageHostLabels         = "storage.zcloud.cn/storagetype"
	StorageNFSHostLabelsValue = "Nfs"
	NFSHostMonitPath          = "/var/lib/singlecloud/nfs-export"
)

func SizetoGb(q resource.Quantity) string {
	return ByteToGb(uint64(q.Value()))
}

func ByteToGb(num uint64) string {
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func KbyteToGb(num int64) string {
	f := float64(num) / (1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func CountFreeSize(t string, u int64) string {
	t1, _ := strconv.ParseFloat(t, 64)
	f1 := float64(u) / (1024 * 1024)
	f := t1 - f1
	if f < 0 {
		f = 0
	}
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func GetNFSPVSize(pv types.PV, mountpoints map[string][]int64) (string, string) {
	var uSize, fSize string
	path := NFSHostMonitPath + "/" + pv.Name
	v, ok := mountpoints[path]
	if ok {
		uSize = KbyteToGb(v[1])
		fSize = CountFreeSize(pv.Size, v[1])
	}
	return uSize, fSize
}

func GetPVSize(pv types.PV, mountpoints map[string][]int64) (string, string) {
	var uSize, fSize string
	for k, v := range mountpoints {
		if strings.Contains(k, pv.Name) {
			uSize = KbyteToGb(v[1])
			fSize = CountFreeSize(pv.Size, v[1])
		}
	}
	return uSize, fSize
}

func GetNodes() (corev1.NodeList, error) {
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

func GetAllPvUsedSize(nodeAgentMgr *nodeagent.NodeAgentManager) (map[string][]int64, error) {
	infos := make(map[string][]int64)
	nodes, err := GetNodes()
	if err != nil {
		return infos, err
	}
	for _, n := range nodes.Items {
		agent, ok := nodeAgentMgr.GetNodeAgent(n.Name)
		if !ok {
			log.Warnf("Get node agent %s failed", n.Name)
			continue
		}
		cli, err := nodeclient.NewClient(agent.Address, 10*time.Second)
		if err != nil {
			log.Warnf("Create node agent client: %s failed: %s", agent.Address, err.Error())
			continue
		}
		mreq := pb.GetMountpointsSizeRequest{}
		mreply, err := cli.GetMountpointsSize(context.TODO(), &mreq)
		if err != nil {
			log.Warnf("Get MountpointsSize on %s failed: %s", agent.Address, err.Error())
			continue
		}
		for k, v := range mreply.Infos {
			if !strings.Contains(k, "pvc-") && !strings.Contains(k, "nfs-") {
				continue
			}
			infos[k] = []int64{v.Tsize, v.Usize, v.Fsize}
		}

		if n.Labels[StorageHostLabels] != StorageNFSHostLabelsValue {
			continue
		}
		dreq := pb.GetDirectorySizeRequest{
			Path: NFSHostMonitPath,
		}
		dreply, err := cli.GetDirectorySize(context.TODO(), &dreq)
		if err != nil {
			log.Warnf("Get DirectorySize on %s failed: %s", agent.Address, err.Error())
			continue
		}
		for k, v := range dreply.Infos {
			infos[k] = []int64{int64(0), v, int64(0)}
		}
	}
	return infos, err
}

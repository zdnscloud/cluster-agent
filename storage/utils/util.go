package utils

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	nodeclient "github.com/zdnscloud/node-agent/client"
	pb "github.com/zdnscloud/node-agent/proto"
)

const (
	StorageHostLabels = "storage.zcloud.cn/storagetype"
	StorageNamespace  = "zcloud"
)

func SizetoGb(q resource.Quantity) string {
	num := uint64(q.Value())
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func KbyteToGb(num int64) string {
	f := float64(num) / (1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func sToi(size string) int64 {
	num, _ := strconv.ParseInt(size, 10, 64)
	return num
}

func byteToGb(num int64) string {
	f := float64(num) / (1024 * 1024 * 1024)
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

func GetNodes() (corev1.NodeList, error) {
	nodes := corev1.NodeList{}
	config, err := config.GetConfig()
	if err != nil {
		return nodes, err
	}
	cli, err := client.New(config, client.Options{})
	if err != nil {
		return nodes, err
	}
	if err = cli.List(context.TODO(), nil, &nodes); err != nil {
		return nodes, err
	}
	return nodes, nil
}

func GetPv(name string) (*corev1.PersistentVolume, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	cli, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}
	pv := &corev1.PersistentVolume{}
	if err = cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, pv); err != nil {
		return nil, err
	}
	return pv, nil
}

func GetNodeForLvmPv(name string) (string, error) {
	config, err := config.GetConfig()
	if err != nil {
		return "", err
	}
	cli, err := client.New(config, client.Options{})
	if err != nil {
		return "", err
	}
	pv := corev1.PersistentVolume{}
	if err = cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &pv); err != nil {
		return "", err
	}
	if pv.Spec.NodeAffinity != nil && pv.Spec.NodeAffinity.Required != nil && pv.Spec.NodeAffinity.Required.NodeSelectorTerms != nil {
		for _, v := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
			for _, i := range v.MatchExpressions {
				if i.Key == "kubernetes.io/hostname" && i.Operator == "In" {
					return i.Values[0], nil
				}
			}
		}
	}
	return "", nil
}

func GetAllPvUsedSize(nodeAgentMgr *nodeagent.NodeAgentManager) (map[string][]int64, error) {
	infos := make(map[string][]int64)
	nodes := nodeAgentMgr.GetNodeAgents()
	for _, node := range nodes {
		cli, err := nodeclient.NewClient(node.Address, 10*time.Second)
		if err != nil {
			log.Warnf("Create node agent client: %s failed: %s", node.Name, err.Error())
			continue
		}
		log.Infof("Get node %s MountpointsSize info", node.Name)
		mreq := pb.GetMountpointsSizeRequest{}
		mreply, err := cli.GetMountpointsSize(context.TODO(), &mreq)
		if err != nil {
			log.Warnf("Get MountpointsSize on %s failed: %s", node.Name, err.Error())
			continue
		}
		for k, v := range mreply.Infos {
			if !strings.Contains(k, "pvc-") && !strings.Contains(k, "nfs-") {
				continue
			}
			infos[k] = []int64{v.Tsize, v.Usize, v.Fsize}
		}
	}
	return infos, nil
}

func GetPVSize(pv *types.PV, mountpoints map[string][]int64) (string, string) {
	var uSize, fSize string
	for k, v := range mountpoints {
		if strings.Contains(k, pv.Name) {
			uSize = KbyteToGb(v[1])
			fSize = CountFreeSize(pv.Size, v[1])
		}
	}
	return uSize, fSize
}

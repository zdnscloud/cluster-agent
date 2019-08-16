package blockdevice

import (
	"context"
	"fmt"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	nodeclient "github.com/zdnscloud/node-agent/client"
	pb "github.com/zdnscloud/node-agent/proto"
	"math"
	"sort"
	"strconv"
	"time"
)

type blockDeviceMgr struct {
	api.DefaultHandler
	NodeAgentMgr *nodeagent.NodeAgentManager
}

func New(nodeAgentMgr *nodeagent.NodeAgentManager) (*blockDeviceMgr, error) {
	return &blockDeviceMgr{
		NodeAgentMgr: nodeAgentMgr,
	}, nil
}

func (m *blockDeviceMgr) RegisterSchemas(version *resttypes.APIVersion, schemas *resttypes.Schemas) {
	schemas.MustImportAndCustomize(version, BlockDevice{}, m, SetBlockDeviceSchema)
}

func (m *blockDeviceMgr) List(ctx *resttypes.Context) interface{} {
	var res BlockDevices
	//res := make([]BlockDevice, 0)
	nodes := m.NodeAgentMgr.GetNodeAgents()
	for _, node := range nodes {
		cli, err := nodeclient.NewClient(node.Address, 10*time.Second)
		if err != nil {
			log.Warnf("Create node agent client: %s failed: %s", node.Name, err.Error())
			continue
		}
		req := pb.GetDisksInfoRequest{}
		reply, err := cli.GetDisksInfo(context.TODO(), &req)
		if err != nil {
			log.Warnf("Get node %s Disk info failed: %s", node.Name, err.Error())
			continue
		}
		var devs Devs
		for k, v := range reply.Infos {
			dev := Dev{
				Name:       k,
				Size:       byteToG(v.Diskinfo["Size"]),
				Parted:     sTob(v.Diskinfo["Parted"]),
				Filesystem: sTob(v.Diskinfo["Filesystem"]),
				Mount:      sTob(v.Diskinfo["Mountpoint"]),
			}
			devs = append(devs, dev)
		}
		sort.Sort(devs)
		host := BlockDevice{
			NodeName:     node.Name,
			BlockDevices: devs,
		}
		res = append(res, host)
	}
	sort.Sort(res)
	return res
}

func byteToG(size string) string {
	num, _ := strconv.ParseInt(size, 10, 64)
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func sTob(str string) bool {
	var res bool
	if str == "true" {
		res = true
	}
	return res
}

package blockinfo

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
	"strconv"
	"time"
)

type blockinfoMgr struct {
	api.DefaultHandler
	NodeAgentMgr *nodeagent.NodeAgentManager
}

func New(nodeAgentMgr *nodeagent.NodeAgentManager) (*blockinfoMgr, error) {
	return &blockinfoMgr{
		NodeAgentMgr: nodeAgentMgr,
	}, nil
}

func (m *blockinfoMgr) RegisterSchemas(version *resttypes.APIVersion, schemas *resttypes.Schemas) {
	schemas.MustImportAndCustomize(version, BlockInfo{}, m, SetBlockInfoSchema)
}

func (m *blockinfoMgr) List(ctx *resttypes.Context) interface{} {
	var res BlockInfo
	nodes := m.NodeAgentMgr.GetNodeAgents()
	for _, node := range nodes {
		var host Host
		host.NodeName = node.Name
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
		for k, v := range reply.Infos {
			dev := Dev{
				Name:       k,
				Size:       byteToG(v.Diskinfo["Size"]),
				Parted:     sTob(v.Diskinfo["Parted"]),
				Filesystem: sTob(v.Diskinfo["Filesystem"]),
				Mount:      sTob(v.Diskinfo["Mountpoint"]),
			}
			host.BlockDevices = append(host.BlockDevices, dev)
		}
		res.Hosts = append(res.Hosts, host)
	}
	return []*BlockInfo{&res}
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

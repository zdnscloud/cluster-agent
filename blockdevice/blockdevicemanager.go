package blockdevice

import (
	"context"
	"fmt"
	cementcache "github.com/zdnscloud/cement/cache"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/nodeagent"
	"github.com/zdnscloud/gorest/resource"
	nodeclient "github.com/zdnscloud/node-agent/client"
	pb "github.com/zdnscloud/node-agent/proto"
	"math"
	"sort"
	"strconv"
	"time"
)

type blockDeviceMgr struct {
	NodeAgentMgr *nodeagent.NodeAgentManager
	cache        *cementcache.Cache
	timeout      int
}

func New(to int, nodeAgentMgr *nodeagent.NodeAgentManager) (*blockDeviceMgr, error) {
	return &blockDeviceMgr{
		NodeAgentMgr: nodeAgentMgr,
		cache:        cementcache.New(1, hashBlockdevices, false),
		timeout:      to,
	}, nil
}

func (m *blockDeviceMgr) RegisterSchemas(version *resource.APIVersion, schemas resource.SchemaManager) {
	schemas.MustImport(version, BlockDevice{}, m)
}

func (m *blockDeviceMgr) List(ctx *resource.Context) interface{} {
	bs := m.GetBuf()
	if len(bs) == 0 {
		bs = m.SetBuf()
	}
	return bs
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

var key = cementcache.HashString("1")

func hashBlockdevices(s cementcache.Value) cementcache.Key {
	return key
}

func (m *blockDeviceMgr) SetBuf() []*BlockDevice {
	bs := m.getBlockdevicesFronNodeAgent()
	if len(bs) == 0 {
		log.Warnf("Has no blockdevices to cache")
		return bs
	}
	m.cache.Add(&bs, time.Duration(m.timeout)*time.Second)
	return bs
}

func (m *blockDeviceMgr) GetBuf() []*BlockDevice {
	res, has := m.cache.Get(key)
	if !has {
		log.Warnf("Cache not found blockdevice")
		return []*BlockDevice{}
	}
	return *res.(*[]*BlockDevice)
}

func (m *blockDeviceMgr) getBlockdevicesFronNodeAgent() []*BlockDevice {
	var res BlockDevices
	nodes := m.NodeAgentMgr.GetNodeAgents()
	for _, node := range nodes {
		cli, err := nodeclient.NewClient(node.Address, 10*time.Second)
		if err != nil {
			log.Warnf("Create node agent client: %s failed: %s", node.Name, err.Error())
			if err := nodeagent.CreateEvent(node.Name, err); err != nil {
				log.Warnf("create event failed: %s", err.Error())
			}
			continue
		}
		defer cli.Close()
		req := pb.GetDisksInfoRequest{}
		reply, err := cli.GetDisksInfo(context.TODO(), &req)
		if err != nil {
			log.Warnf("Get node %s Disk info failed: %s", node.Name, err.Error())
			if err := nodeagent.CreateEvent(node.Name, err); err != nil {
				log.Warnf("create event failed: %s", err.Error())
			}
			continue
		}
		var devs Devs
		for k, v := range reply.Disks {
			dev := Dev{
				Name:       k,
				Size:       byteToG(strconv.Itoa(int(v.Size))),
				Parted:     v.Parted,
				Filesystem: v.Filesystem,
				Mount:      v.Mountpoint,
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
	bs := make([]*BlockDevice, len(res))
	for i := 0; i < len(res); i++ {
		bs[i] = &res[i]
	}
	return bs
}

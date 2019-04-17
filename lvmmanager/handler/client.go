package handler

import (
	"context"
	lvmd "github.com/google/lvmd/proto"
	"github.com/zdnscloud/cluster-agent/lvmmanager/lvmdclient"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"time"
)

func conApi() (client.Client, error) {
	config, err := config.GetConfig()
	if err != nil {
		Fatal("get k8s config failed:%s", err.Error())
		return nil, err
	}
	cli, err := client.New(config, client.Options{})
	if err != nil {
		Fatal("create k8s client failed:%s", err.Error())
		return nil, err
	}
	return cli, nil
}

func GetStorage() ([]string, error) {
	cli, err := conApi()
	if err != nil {
		return nil, err
	}
	storages := storagev1.StorageClassList{}
	cli.List(context.TODO(), nil, &storages)
	var res []string
	for _, s := range storages.Items {
		res = append(res, s.Name)
	}
	return res, nil

}

func GetNode() ([]corev1.Node, error) {
	cli, err := conApi()
	if err != nil {
		return nil, err
	}
	nodes := corev1.NodeList{}
	cli.List(context.TODO(), nil, &nodes)
	var res []corev1.Node
	for _, n := range nodes.Items {
		v, ok := n.Labels[ZkeStorageLabel]
		if ok && v == "true" {
			res = append(res, n)
		}
	}
	return res, nil
}

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
func GetLVM(node string, vgName string) ([]*lvmd.LogicalVolume, error) {
	addr := node + ":" + LvmdPort
	tmout := time.Second * ConTimeout
	conn, err := lvmdclient.NewLVMConnection(addr, tmout)
	defer conn.Close()
	if err != nil {
		return nil, err
	}
	ctx := context.TODO()
	resp, err := conn.GetLV(ctx, vgName)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func getStorageSlice() []Storage {
	var storages []Storage
	var tsize uint64
	var fszie uint64
	storageclasss, err := GetStorage()
	if err != nil {
		return nil
	}
	for _, s := range storageclasss {
		var storage Storage
		storage.Name = s
		if s == DefaultStorageClass {
			nodes := getNodeSlice()
			for _, n := range nodes {
				fszie += n.FreeSize
				tsize += n.TotalSize
			}
			storage.TotalSize = tsize
			storage.FreeSize = fszie
			//		storage.StorageNode = nodes
		}
		storages = append(storages, storage)
	}
	return storages
}

func getNodeSlice() []Node {
	var nodes []Node
	noderes, err := GetNode()
	if err != nil {
		return nil
	}
	for _, v := range noderes {
		node := Node{
			Name: v.Name,
			Addr: v.Annotations[ZkeInternalIPAnnKey],
		}
		vgs := getVgSlice(node.Addr)
		//node.Vgs = vgs
		for _, v := range vgs {
			if v.Name == DefaultVgName {
				node.TotalSize = v.Size
				node.FreeSize = v.FreeSize
			}
		}
		node.SetID(node.Name)
		nodes = append(nodes, node)
	}
	return nodes
}

func getVgSlice(node string) []VG {
	var vgs []VG
	vgres, err := GetVG(node)
	if err != nil {
		return nil
	}
	for _, v := range vgres {
		if v.Name == DefaultVgName {
			vg := VG{
				Name:     v.Name,
				Size:     v.Size,
				Uuid:     v.Uuid,
				Tags:     v.Tags,
				FreeSize: v.FreeSize,
			}
			//	lvms := getLvmSlice(node, vg.Name)
			//	vg.Lvms = lvms
			vg.SetID(vg.Name)
			vgs = append(vgs, vg)
		}
	}
	return vgs
}

func getLvmSlice(node string, vg string) []LVM {
	var lvms []LVM
	lvmres, err := GetLVM(node, vg)
	if err != nil {
		return nil
	}
	for _, v := range lvmres {
		lvm := LVM{
			Name: v.Name,
			Size: v.Size,
			Uuid: v.Uuid,
			Tags: v.Tags,
		}
		lvm.SetID(lvm.Name)
		lvms = append(lvms, lvm)
	}
	return lvms
}

func nameToAddr(name string) string {
	nodestmp := getNodeSlice()
	for _, v := range nodestmp {
		if name == v.Name {
			return v.Addr
		} else {
			continue
		}
	}
	return ""
}

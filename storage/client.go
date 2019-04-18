package storage

import (
	"fmt"
	"strconv"
)

func getStorages() []Storage {
	var storages []Storage
	storageclasss, err := GetStorageClass()
	if err != nil {
		return nil
	}
	for _, c := range storageclasss {
		var storage Storage
		if c == LocalStorageClass {
			storage = getLocalStorage(c, storage)
		}
		storages = append(storages, storage)
	}
	return storages
}

func getLocalStorage(classname string, storage Storage) Storage {
	var tsize int
	var fszie int
	nodes := getNodes()
	for _, n := range nodes {
		fszie += n.FreeSize
		tsize += n.TotalSize
	}
	storage.Name = classname
	storage.TotalSize = tsize
	storage.FreeSize = fszie
	storage.Nodes = nodes
	storage.SetID(storage.Name)
	return storage
}

func getNodes() []Node {
	var nodes []Node
	noderes, err := GetNode()
	if err != nil {
		return nil
	}
	for _, v := range noderes {
		addr := v.Annotations[ZkeInternalIPAnnKey]
		node := Node{
			Name: v.Name,
		}
		vgs := getVgs(addr)
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

func getVgs(node string) []VG {
	var vgs []VG
	vgres, err := GetVG(node)
	if err != nil {
		return nil
	}
	for _, v := range vgres {
		if v.Name == DefaultVgName {
			vg := VG{
				Name:     v.Name,
				Size:     byteToGb(v.Size),
				Uuid:     v.Uuid,
				Tags:     v.Tags,
				FreeSize: byteToGb(v.FreeSize),
			}
			vg.SetID(vg.Name)
			vgs = append(vgs, vg)
		}
	}
	return vgs
}

func nameToAddr(name string) string {
	nodestmp := getNodes()
	for _, v := range nodestmp {
		if name == v.Name {
			return v.Addr
		} else {
			continue
		}
	}
	return ""
}

func byteToGb(num uint64) int {
	f := float64(num) / 1024 / 1024 / 1024
	i, _ := strconv.Atoi(fmt.Sprintf("%.0f", f))
	return i
}

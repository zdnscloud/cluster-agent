package storage

import (
	"fmt"
	"github.com/zdnscloud/gok8s/cache"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

type StorageCache struct {
	storages map[string]*Storage
	cache    cache.Cache
}

func newStorageCache(cache cache.Cache) *StorageCache {
	return &StorageCache{
		cache:    cache,
		storages: make(map[string]*Storage),
	}
}

func (sc *StorageCache) GetStorages() []Storage {
	var storages []Storage
	for _, storage := range sc.storages {
		storages = append(storages, *storage)
	}
	return storages
}

func (sc *StorageCache) OnNewNode(k8snode *corev1.Node) {
	v, ok := k8snode.Labels[ZkeStorageLabel]
	if ok && v == "true" {
		nodes := sc.storages["lvm"].Nodes
		node := getLvmNode(k8snode)
		nodes = append(nodes, node)
		var tsize int
		var fsize int
		for _, n := range nodes {
			fsize += n.FreeSize
			tsize += n.TotalSize
		}
		sc.storages["lvm"] = &Storage{
			Nodes:     nodes,
			Name:      "lvm",
			TotalSize: tsize,
			FreeSize:  fsize,
		}
		//fmt.Println("StorageNode:", k8snode.Annotations["zdnscloud.cn/internal-ip"])
	}
}

func (sc *StorageCache) OnNewStorageClass(k8ssc *storagev1.StorageClass) {
	//fmt.Println("######################")
	//fmt.Println(k8ssc.Name)
	//fmt.Println(k8ssc)
	sc.storages[k8ssc.Name] = &Storage{
		Name: k8ssc.Name,
	}
}

func (sc *StorageCache) OnNewPV(k8spv *corev1.PersistentVolume) {
	fmt.Println(k8spv.Name)
}

func (sc *StorageCache) OnDelStorageClass(k8ssc *storagev1.StorageClass) {
	fmt.Println(k8ssc.Name)
	delete(sc.storages, k8ssc.Name)
}

func getLvmNode(k8snode *corev1.Node) Node {
	addr := k8snode.Annotations[ZkeInternalIPAnnKey]
	vg := getVg(addr)
	return Node{
		Name:      k8snode.Name,
		TotalSize: vg.Size,
		FreeSize:  vg.FreeSize,
	}
}

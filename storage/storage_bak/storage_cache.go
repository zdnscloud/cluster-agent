package storage

import (
	"fmt"
	"github.com/zdnscloud/gok8s/cache"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"unsafe"
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
		tsize := sc.storages["lvm"].TotalSize
		fsize := sc.storages["lvm"].FreeSize
		num := len(nodes)
		flag := 1
		if num == 0 {
			flag = 0
		} else {
			for _, n := range nodes {
				if k8snode.Name == n.Name {
					flag = 1
					break
				} else {
					flag = 0
					continue
				}
			}
		}
		if flag == 0 {
			node := getLvmNode(k8snode)
			nodes = append(nodes, node)
			fsize += node.FreeSize
			tsize += node.TotalSize
			sc.storages["lvm"].Nodes = nodes
			sc.storages["lvm"].TotalSize = tsize
			sc.storages["lvm"].FreeSize = fsize
		}
	}
}

func (sc *StorageCache) OnInit(k8ssc *storagev1.StorageClass) {
	sc.storages[k8ssc.Name] = &Storage{
		Name: k8ssc.Name,
	}
}
func (sc *StorageCache) OnNewStorageClass(k8ssc *storagev1.StorageClass) {
	sc.storages[k8ssc.Name].Name = k8ssc.Name
}

func (sc *StorageCache) OnNewPV(k8spv *corev1.PersistentVolume) {
	quantity := k8spv.Spec.Capacity["storage"]
	int64value := quantity.Value()
	uint64value := (*uint64)(unsafe.Pointer(&int64value))
	pvsize := byteToGb(*uint64value)
	v, ok := k8spv.Labels["test"]
	if ok && v == "nfs" {
		sc.storages["nfs"].TotalSize = pvsize
		sc.storages["nfs"].FreeSize = pvsize
	}
	stc := k8spv.Spec.StorageClassName
	pvs := sc.storages[stc].PVs
	num := len(pvs)
	flag := 1
	if num == 0 {
		flag = 0
	} else {
		for _, p := range pvs {
			if k8spv.Name == p.Name {
				flag = 1
				break
			} else {
				flag = 0
				continue
			}
		}
	}
	if flag == 0 {
		pv := Pv{
			Name: k8spv.Name,
			Size: pvsize,
			Pvc:  k8spv.Spec.ClaimRef.Name,
		}
		pvs = append(sc.storages[stc].PVs, pv)
	}
	sc.storages[stc].PVs = pvs
}

func (sc *StorageCache) OnNewPod(k8spod *corev1.Pod) {
	pvc := k8spod.Spec.Volumes
	for _, v := range pvc {
		if v.PersistentVolumeClaim != nil {
			for _, m := range sc.storages {
				for i, p := range m.PVs {
					if v.PersistentVolumeClaim.ClaimName == p.Pvc {
						pods := m.PVs[i].Pods
						flag := 1
						num := len(pods)
						if num == 0 {
							flag = 0
						} else {
							for _, p := range pods {
								if k8spod.Name == p {
									flag = 1
									break
								} else {
									flag = 0
									continue
								}
							}
						}
						if flag == 0 {
							pods = append(pods, k8spod.Name)
						}
						m.PVs[i].Pods = pods
					}
				}
			}
		}
	}
}

func (sc *StorageCache) OnDelNode(k8snode *corev1.Node) {
	v, ok := k8snode.Labels[ZkeStorageLabel]
	if ok && v == "true" {
		for i, v := range sc.storages["lvm"].Nodes {
			if v.Name == k8snode.Name {
				fmt.Println("######", k8snode.Name)
				sc.storages["lvm"].Nodes = append(sc.storages["lvm"].Nodes[:i], sc.storages["lvm"].Nodes[i+1:]...)
				break
			}
		}
	}
}

func (sc *StorageCache) OnDelStorageClass(k8ssc *storagev1.StorageClass) {
	//fmt.Println(k8ssc.Name)
	delete(sc.storages, k8ssc.Name)
}

func (sc *StorageCache) OnDelPV(k8spv *corev1.PersistentVolume) {
	//fmt.Println(k8spv.Name)
	sc.storages["nfs"] = &Storage{
		Name: "nfs",
	}

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

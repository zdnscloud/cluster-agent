package storage

import (
	"context"
	//"fmt"
	"github.com/zdnscloud/gok8s/cache"
	resttypes "github.com/zdnscloud/gorest/types"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
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

func (sc *StorageCache) GetStorage(ctx *resttypes.Context) []Storage {
	id := ctx.Object.GetID()
	var storages []Storage
	for _, storage := range sc.storages {
		if id == storage.Name {
			storages = append(storages, *storage)
		}
	}
	return storages
}

func (sc *StorageCache) OnInit(k8ssc *storagev1.StorageClass) {
	sc.storages[k8ssc.Name] = &Storage{
		Name: k8ssc.Name,
	}
}

func (sc *StorageCache) OnInitNFS() {
	pvc := corev1.PersistentVolumeClaim{}
	_ = sc.cache.Get(context.TODO(), k8stypes.NamespacedName{ZKEStorageNamespace, ZKENFSPvcName}, &pvc)
	quantity := pvc.Spec.Resources.Requests["storage"]
	pvsize := sizetog(quantity)
	sc.storages[NFS].TotalSize = pvsize
	sc.storages[NFS].FreeSize = pvsize
}

func (sc *StorageCache) OnNewStorageClass(k8ssc *storagev1.StorageClass) {
	sc.storages[k8ssc.Name].Name = k8ssc.Name
}

func (sc *StorageCache) OnNewNode(k8snode *corev1.Node) {
	v, ok := k8snode.Labels[ZkeStorageLabel]
	if ok && v == "true" {
		nodes := sc.storages[LVM].Nodes
		tsize := sc.storages[LVM].TotalSize
		fsize := sc.storages[LVM].FreeSize
		flag := isNodeExist(nodes, k8snode)
		if !flag {
			node := getLvmNode(k8snode)
			nodes = append(nodes, node)
			fsize += node.FreeSize
			tsize += node.TotalSize
			sc.storages[LVM].Nodes = nodes
			sc.storages[LVM].TotalSize = tsize
			sc.storages[LVM].FreeSize = fsize
		}
	}
}

func (sc *StorageCache) OnNewPV(k8spv *corev1.PersistentVolume) {
	quantity := k8spv.Spec.Capacity["storage"]
	pvsize := sizetog(quantity)
	stc := k8spv.Spec.StorageClassName
	pvs := sc.storages[stc].PVs
	flag := isPvExist(pvs, k8spv)
	if !flag {
		pv := Pv{
			Name:   k8spv.Name,
			Size:   pvsize,
			Pvc:    k8spv.Spec.ClaimRef.Name,
			Status: k8spv.Status.Phase,
		}
		pvs = append(sc.storages[stc].PVs, pv)
	}
	sc.storages[stc].PVs = pvs
}

func (sc *StorageCache) OnNewPod(k8spod *corev1.Pod) {
	pvc := k8spod.Spec.Volumes
	for _, v := range pvc {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvc := v.PersistentVolumeClaim.ClaimName
		stc := sc.getStc(pvc)
		if stc == "" {
			return
		}
		pvs := sc.storages[stc].PVs
		for i, p := range pvs {
			if pvc == p.Pvc {
				pods := pvs[i].Pods
				flag := isPodExist(pods, k8spod.Name)
				if !flag {
					pvs[i].Pods = append(pods, k8spod.Name)
				}
			}
		}
	}
}

func (sc *StorageCache) OnDelNode(k8snode *corev1.Node) {
	v, ok := k8snode.Labels[ZkeStorageLabel]
	if ok && v == "true" {
		for i, n := range sc.storages[LVM].Nodes {
			if n.Name == k8snode.Name {
				sc.storages[LVM].Nodes = append(sc.storages[LVM].Nodes[:i], sc.storages[LVM].Nodes[i+1:]...)
				break
			}
		}
	}
}

func (sc *StorageCache) OnDelStorageClass(k8ssc *storagev1.StorageClass) {
	delete(sc.storages, k8ssc.Name)
}

func (sc *StorageCache) OnDelPV(k8spv *corev1.PersistentVolume) {
	stc := k8spv.Spec.StorageClassName
	pvs := sc.storages[stc].PVs
	for i, p := range pvs {
		if p.Name == k8spv.Name {
			sc.storages[stc].PVs = append(sc.storages[stc].PVs[:i], sc.storages[stc].PVs[i+1:]...)
			break
		}
	}
}

func (sc *StorageCache) OnDelPod(k8spod *corev1.Pod) {
	pvc := k8spod.Spec.Volumes
	for _, v := range pvc {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvc := v.PersistentVolumeClaim.ClaimName
		stc := sc.getStc(pvc)
		pvs := sc.storages[stc].PVs

		for i, p := range pvs {
			if pvc == p.Pvc {
				for j, d := range p.Pods {
					if d == k8spod.Name {
						pvs[i].Pods = append(pvs[i].Pods[:j], pvs[i].Pods[j+1:]...)
						break
					} else {
						continue
					}
				}
			}
		}
		sc.storages[stc].PVs = pvs
	}
}

func (sc *StorageCache) OnUpdatePV(k8spv *corev1.PersistentVolume) {
	_, ok := sc.storages[k8spv.Spec.StorageClassName]
	if ok == false {
		return
	}
	quantity := k8spv.Spec.Capacity["storage"]
	pvsize := sizetog(quantity)
	pv := Pv{
		Name:   k8spv.Name,
		Size:   pvsize,
		Pvc:    k8spv.Spec.ClaimRef.Name,
		Status: k8spv.Status.Phase,
	}
	stc := k8spv.Spec.StorageClassName
	pvs := sc.storages[stc].PVs
	for i, p := range pvs {
		if k8spv.Name == p.Name {
			sc.storages[stc].PVs[i] = pv
			return
		}
	}
}

func (sc *StorageCache) getStc(p string) string {
	for k, v := range sc.storages {
		for _, i := range v.PVs {
			if i.Pvc == p {
				return k
			}
		}
	}
	return ""
}

func isPodExist(pods []string, pod string) bool {
	flag := true
	num := len(pods)
	if num == 0 {
		flag = false
	} else {
		for _, p := range pods {
			if pod == p {
				flag = true
				break
			} else {
				flag = false
				continue
			}
		}
	}
	return flag
}

func isPvExist(pvs []Pv, pv *corev1.PersistentVolume) bool {
	flag := true
	num := len(pvs)
	if num == 0 {
		flag = false
	} else {
		for _, p := range pvs {
			if pv.Name == p.Name {
				flag = true
				break
			} else {
				flag = false
				continue
			}
		}
	}
	return flag
}
func isNodeExist(nodes []Node, node *corev1.Node) bool {
	flag := true
	num := len(nodes)
	if num == 0 {
		flag = false
	} else {
		for _, n := range nodes {
			if node.Name == n.Name {
				flag = true
				break
			} else {
				flag = false
				continue
			}
		}
	}
	return flag
}

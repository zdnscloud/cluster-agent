package lvm

import (
	"fmt"
	"github.com/zdnscloud/cluster-agent/storage/types"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"unsafe"
)

func (s *Storage) OnNewPV(pv *corev1.PersistentVolume) {
	sc := pv.Spec.StorageClassName
	if sc == "lvm" {

		fmt.Println("add pv:", pv.Name)
		quantity := pv.Spec.Capacity["storage"]
		pvsize := sizetog(quantity)
		//pvc := s.PvAndPvc[pv.Name].Name
		//pods := s.PvcAndPod[pvc]
		p := types.PV{
			Name: pv.Name,
			Size: pvsize,
			//Pods: pods,
		}
		pvs := s.PVs
		s.PVs = append(pvs, p)
	}
}

func (s *Storage) OnNewPvc(pvc *corev1.PersistentVolumeClaim) {
	fmt.Println("add pvc:", pvc.Name)
	pods := s.PvcAndPod[pvc.Name]
	p := types.Pvc{
		Name:         pvc.Name,
		Namespace:    pvc.Namespace,
		StorageClass: *pvc.Spec.StorageClassName,
		VolumeName:   pvc.Spec.VolumeName,
		Pods:         pods,
	}
	s.PvAndPvc[pvc.Spec.VolumeName] = p
}

func (s *Storage) OnDelPvc(pvc *corev1.PersistentVolumeClaim) {
	delete(s.PvAndPvc, pvc.Spec.VolumeName)
}

func (s *Storage) OnNewPod(pod *corev1.Pod) {
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvc := v.PersistentVolumeClaim.ClaimName
		p := types.Pod{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		}
		//fmt.Println(pod.Name, pvc)
		//fmt.Println(s.PvcAndPod)
		flag := 1
		for _, m := range s.PvcAndPod[pvc] {
			if m.Name == pod.Name {
				flag = 0
				break
			} else {
				flag = 1
				continue
			}
		}
		if flag == 1 {
			s.PvcAndPod[pvc] = append(s.PvcAndPod[pvc], p)
		}
		//fmt.Println(s.PvcAndPod)
	}
}

func (s *Storage) OnDelPod(pod *corev1.Pod) {
	fmt.Println("del:", pod.Name)
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvc := v.PersistentVolumeClaim.ClaimName
		_, ok := s.PvcAndPod[pvc]
		if ok {
			//fmt.Println(s.PvcAndPod[pvc])
			for i, n := range s.PvcAndPod[pvc] {
				if n.Name == pod.Name {
					s.PvcAndPod[pvc] = append(s.PvcAndPod[pvc][:i], s.PvcAndPod[pvc][i+1:]...)
					break
				} else {
					continue
				}
			}
		}
	}
}

func (s *Storage) OnUpdatePvc(pvc *corev1.PersistentVolumeClaim) {
	fmt.Println("update pvc:", pvc.Name)
	pods := s.PvcAndPod[pvc.Name]
	p := types.Pvc{
		Name:         pvc.Name,
		Namespace:    pvc.Namespace,
		StorageClass: *pvc.Spec.StorageClassName,
		VolumeName:   pvc.Spec.VolumeName,
		Pods:         pods,
	}
	s.PvAndPvc[pvc.Spec.VolumeName] = p
	return
}

func (s *Storage) OnDelPV(pv *corev1.PersistentVolume) {
	pvs := s.PVs
	for i, v := range pvs {
		if v.Name == pv.Name {
			s.PVs = append(s.PVs[:i], s.PVs[i+1:]...)
		}
	}
}

func sizetog(q resource.Quantity) int {
	int64value := q.Value()
	uint64value := (*uint64)(unsafe.Pointer(&int64value))
	pvsize := byteToGb(*uint64value)
	return pvsize
}

func byteToGb(num uint64) int {
	f := float64(num) / 1024 / 1024 / 1024
	i, _ := strconv.Atoi(fmt.Sprintf("%.0f", f))
	return i
}

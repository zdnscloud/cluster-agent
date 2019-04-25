package storageclass

import (
	"github.com/zdnscloud/cluster-agent/storage/types"
	corev1 "k8s.io/api/core/v1"
)

func (s *StorageCache) OnNewPV(pv *corev1.PersistentVolume) {
	quantity := pv.Spec.Capacity["storage"]
	pvsize := sizetog(quantity)
	p := types.PV{
		Name:             pv.Name,
		Size:             pvsize,
		StorageClassName: pv.Spec.StorageClassName,
	}
	pvs := s.PVs
	s.PVs = append(pvs, p)
}

func (s *StorageCache) OnNewPvc(pvc *corev1.PersistentVolumeClaim) {
	pods := s.PvcAndPod[pvc.Name]
	p := PVC{
		Name:       pvc.Name,
		VolumeName: pvc.Spec.VolumeName,
		Pods:       pods,
	}
	s.PvAndPvc[pvc.Spec.VolumeName] = p
}

func (s *StorageCache) OnNewPod(pod *corev1.Pod) {
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvc := v.PersistentVolumeClaim.ClaimName
		p := types.Pod{
			Name: pod.Name,
		}
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
	}
}

func (s *StorageCache) OnDelPV(pv *corev1.PersistentVolume) {
	pvs := s.PVs
	for i, v := range pvs {
		if v.Name == pv.Name {
			s.PVs = append(s.PVs[:i], s.PVs[i+1:]...)
		}
	}
}

func (s *StorageCache) OnDelPvc(pvc *corev1.PersistentVolumeClaim) {
	delete(s.PvAndPvc, pvc.Spec.VolumeName)
}

func (s *StorageCache) OnDelPod(pod *corev1.Pod) {
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvc := v.PersistentVolumeClaim.ClaimName
		_, ok := s.PvcAndPod[pvc]
		if ok {
			for i, n := range s.PvcAndPod[pvc] {
				if n.Name == pod.Name {
					s.PvcAndPod[pvc] = append(s.PvcAndPod[pvc][:i], s.PvcAndPod[pvc][i+1:]...)
					if len(s.PvcAndPod[pvc]) == 0 {
						s.PvcAndPod[pvc] = nil
					}
					break
				} else {
					continue
				}
			}
		}
	}
}

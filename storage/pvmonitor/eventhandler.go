package pvmonitor

import (
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	corev1 "k8s.io/api/core/v1"
)

func (s *PVMonitor) OnNewPV(pv *corev1.PersistentVolume) {
	if pv.Spec.StorageClassName != s.StorageClassName {
		return
	}
	quantity := pv.Spec.Capacity["storage"]
	pvsize := utils.SizetoGb(quantity)
	p := types.PV{
		Name:             pv.Name,
		Size:             pvsize,
		StorageClassName: pv.Spec.StorageClassName,
	}
	pvs := s.PVs
	s.PVs = append(pvs, p)
}

func (s *PVMonitor) OnNewPVC(pvc *corev1.PersistentVolumeClaim) {
	cls := *pvc.Spec.StorageClassName
	if cls != s.StorageClassName {
		return
	}
	pvcns := pvc.Namespace + "/" + pvc.Name
	p := PVC{
		Name:           pvc.Name,
		NamespacedName: pvcns,
	}
	s.PvAndPVC[pvc.Spec.VolumeName] = p
}

func (s *PVMonitor) OnNewPod(pod *corev1.Pod) {
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvcName := v.PersistentVolumeClaim.ClaimName
		hasPod := false
		for _, m := range s.PVCAndPod[pvcName] {
			if m.Name == pod.Name {
				hasPod = true
				break
			}
		}
		p := types.Pod{
			Name: pod.Name,
		}
		if !hasPod {
			s.PVCAndPod[pvcName] = append(s.PVCAndPod[pvcName], p)
		}
	}
}

func (s *PVMonitor) OnDelPV(pv *corev1.PersistentVolume) {
	if pv.Spec.StorageClassName != s.StorageClassName {
		return
	}
	pvs := s.PVs
	for i, v := range pvs {
		if v.Name == pv.Name {
			s.PVs = append(s.PVs[:i], s.PVs[i+1:]...)
		}
	}
}

func (s *PVMonitor) OnDelPVC(pvc *corev1.PersistentVolumeClaim) {
	cls := *pvc.Spec.StorageClassName
	if cls != s.StorageClassName {
		return
	}
	delete(s.PvAndPVC, pvc.Spec.VolumeName)
}

func (s *PVMonitor) OnDelPod(pod *corev1.Pod) {
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvcName := v.PersistentVolumeClaim.ClaimName
		for i, p := range s.PVCAndPod[pvcName] {
			if p.Name == pod.Name {
				s.PVCAndPod[pvcName] = append(s.PVCAndPod[pvcName][:i], s.PVCAndPod[pvcName][i+1:]...)
				if len(s.PVCAndPod[pvcName]) == 0 {
					delete(s.PVCAndPod, pvcName)
				}
				break
			}
		}
	}
}

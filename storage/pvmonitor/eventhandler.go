package pvmonitor

import (
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	corev1 "k8s.io/api/core/v1"
)

func (s *PVMonitor) OnNewPV(pv *corev1.PersistentVolume) {
	if pv.Spec.StorageClassName != s.Name {
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

func (s *PVMonitor) OnNewPvc(pvc *corev1.PersistentVolumeClaim) {
	cls := *pvc.Spec.StorageClassName
	if cls != s.Name {
		return
	}
	pods := s.PvcAndPod[pvc.Name]
	p := PVC{
		Name:       pvc.Name,
		VolumeName: pvc.Spec.VolumeName,
		Pods:       pods,
	}
	s.PvAndPvc[pvc.Spec.VolumeName] = p
	s.PVCAndSc[pvc.Name] = cls
}

func (s *PVMonitor) OnNewPod(pod *corev1.Pod) {
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvcName := v.PersistentVolumeClaim.ClaimName
		cls, ok := s.PVCAndSc[pvcName]
		if !ok {
			return
		}
		if cls != s.Name {
			return
		}
		hasPod := false
		for _, m := range s.PvcAndPod[pvcName] {
			if m.Name == pod.Name {
				hasPod = true
				break
			}
		}
		p := types.Pod{
			Name: pod.Name,
		}
		if !hasPod {
			s.PvcAndPod[pvcName] = append(s.PvcAndPod[pvcName], p)
		}
	}
}

func (s *PVMonitor) OnDelPV(pv *corev1.PersistentVolume) {
	if pv.Spec.StorageClassName != s.Name {
		return
	}
	pvs := s.PVs
	for i, v := range pvs {
		if v.Name == pv.Name {
			s.PVs = append(s.PVs[:i], s.PVs[i+1:]...)
		}
	}
}

func (s *PVMonitor) OnDelPvc(pvc *corev1.PersistentVolumeClaim) {
	cls := *pvc.Spec.StorageClassName
	if cls != s.Name {
		return
	}
	delete(s.PvAndPvc, pvc.Spec.VolumeName)
}

func (s *PVMonitor) OnDelPod(pod *corev1.Pod) {
	vs := pod.Spec.Volumes
	for _, v := range vs {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		pvcName := v.PersistentVolumeClaim.ClaimName
		cls, ok := s.PVCAndSc[pvcName]
		if !ok {
			return
		}
		if cls != s.Name {
			return
		}
		for i, p := range s.PvcAndPod[pvcName] {
			if p.Name == pod.Name {
				s.PvcAndPod[pvcName] = append(s.PvcAndPod[pvcName][:i], s.PvcAndPod[pvcName][i+1:]...)
				if len(s.PvcAndPod[pvcName]) == 0 {
					delete(s.PvcAndPod, pvcName)
				}
				break
			}
		}
	}
}

package pvmonitor

import (
	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
	corev1 "k8s.io/api/core/v1"
)

const (
	CSIAnnotations = "volume.beta.kubernetes.io/storage-provisioner"
)

func (s *PVMonitor) OnNewPV(pv *corev1.PersistentVolume) {
	if pv.Spec.PersistentVolumeSource.CSI == nil || pv.Spec.PersistentVolumeSource.CSI.Driver != s.DriverName {
		return
	}
	pvsize := utils.SizetoGb(pv.Spec.Capacity["storage"])
	p := types.PV{
		Name:             pv.Name,
		Size:             pvsize,
		StorageClassName: pv.Spec.StorageClassName,
	}
	s.PVs = append(s.PVs, p)
}

func (s *PVMonitor) OnUpdatePV(pv *corev1.PersistentVolume) {
	for i, p := range s.PVs {
		if p.Name != pv.Name {
			continue
		}
		pvsize := utils.SizetoGb(pv.Spec.Capacity["storage"])
		if p.Size == pvsize {
			continue
		}
		s.PVs[i].Size = pvsize
	}
}

func (s *PVMonitor) OnNewPVC(pvc *corev1.PersistentVolumeClaim) {
	if pvc.Spec.StorageClassName == nil || pvc.Annotations[CSIAnnotations] != s.DriverName {
		return
	}
	p := PVC{
		Name:           pvc.Name,
		NamespacedName: pvc.Namespace + "/" + pvc.Name,
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
	if pv.Spec.PersistentVolumeSource.CSI == nil || pv.Spec.PersistentVolumeSource.CSI.Driver != s.DriverName {
		return
	}
	for i, v := range s.PVs {
		if v.Name == pv.Name {
			s.PVs = append(s.PVs[:i], s.PVs[i+1:]...)
		}
	}
}

func (s *PVMonitor) OnDelPVC(pvc *corev1.PersistentVolumeClaim) {
	if pvc.Spec.StorageClassName == nil || pvc.Annotations[CSIAnnotations] != s.DriverName {
		return
	}
	delete(s.PvAndPVC, pvc.Spec.VolumeName)
}

func (s *PVMonitor) OnDelPod(pod *corev1.Pod) {
	for _, v := range pod.Spec.Volumes {
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

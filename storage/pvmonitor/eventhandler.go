package pvmonitor

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/zdnscloud/cluster-agent/storage/types"
	"github.com/zdnscloud/cluster-agent/storage/utils"
)

func (s *PVMonitor) OnNewPV(pv *corev1.PersistentVolume) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.Pvs[pv.Name]; ok {
		return
	}

	s.Pvs[pv.Name] = &types.PV{
		Name:             pv.Name,
		Size:             utils.SizetoGb(pv.Spec.Capacity["storage"]),
		StorageClassName: pv.Spec.StorageClassName,
		Node:             getNodeIfHas(pv),
	}
}

func (s *PVMonitor) OnUpdatePV(pv *corev1.PersistentVolume) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if p, ok := s.Pvs[pv.Name]; ok {
		p.Size = utils.SizetoGb(pv.Spec.Capacity["storage"])
	}
}

func (s *PVMonitor) OnDelPV(pv *corev1.PersistentVolume) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.Pvs, pv.Name)
}

func (s *PVMonitor) OnNewPVC(pvc *corev1.PersistentVolumeClaim) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Pvcs[pvc.Spec.VolumeName] = pvc.Name
}

func (s *PVMonitor) OnDelPVC(pvc *corev1.PersistentVolumeClaim) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.Pvcs, pvc.Spec.VolumeName)
}

func (s *PVMonitor) OnNewPod(pod *corev1.Pod) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		if _, ok := s.Pods[v.PersistentVolumeClaim.ClaimName]; !ok {
			s.Pods[v.PersistentVolumeClaim.ClaimName] = make([]types.Pod, 0)
		}
		if isPodExist(s.Pods[v.PersistentVolumeClaim.ClaimName], pod.Name) == false {
			s.Pods[v.PersistentVolumeClaim.ClaimName] = append(s.Pods[v.PersistentVolumeClaim.ClaimName], types.Pod{pod.Name})
		}
	}
}

func (s *PVMonitor) OnDelPod(pod *corev1.Pod) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, v := range pod.Spec.Volumes {
		if v.PersistentVolumeClaim == nil {
			continue
		}
		if pods, ok := s.Pods[v.PersistentVolumeClaim.ClaimName]; ok {
			for i, _pod := range pods {
				if _pod.Name == pod.Name {
					s.Pods[v.PersistentVolumeClaim.ClaimName] = append(s.Pods[v.PersistentVolumeClaim.ClaimName][:i], s.Pods[v.PersistentVolumeClaim.ClaimName][i+1:]...)
				}
			}
		}
	}
}

func isPodExist(pods []types.Pod, name string) bool {
	for _, pod := range pods {
		if pod.Name == name {
			return true
		}
	}
	return false
}

func getNodeIfHas(pv *corev1.PersistentVolume) string {
	if pv.Spec.NodeAffinity != nil && pv.Spec.NodeAffinity.Required != nil && pv.Spec.NodeAffinity.Required.NodeSelectorTerms != nil {
		for _, v := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
			for _, i := range v.MatchExpressions {
				if i.Key == "kubernetes.io/hostname" && i.Operator == "In" {
					return i.Values[0]
				}
			}
		}
	}
	return ""
}

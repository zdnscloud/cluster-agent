package network

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
)

type NetworkCache struct {
	nodeNetworks    map[string]*NodeNetwork
	podNetworks     map[string]*PodNetwork
	serviceNetworks map[string]*ServiceNetwork
}

func newNetworkCache() *NetworkCache {
	return &NetworkCache{
		nodeNetworks:    make(map[string]*NodeNetwork),
		podNetworks:     make(map[string]*PodNetwork),
		serviceNetworks: make(map[string]*ServiceNetwork),
	}
}

func (nc *NetworkCache) GetNodeNetworks() NodeNetworks {
	var nodeNetworks NodeNetworks
	for _, nodeNetwork := range nc.nodeNetworks {
		nodeNetworks = append(nodeNetworks, nodeNetwork)
	}
	sort.Sort(nodeNetworks)
	return nodeNetworks
}

func (nc *NetworkCache) GetPodNetworks() PodNetworks {
	var podNetworks PodNetworks
	for _, podNetwork := range nc.podNetworks {
		podNetworks = append(podNetworks, podNetwork)
	}
	sort.Sort(podNetworks)
	return podNetworks
}

func (nc *NetworkCache) GetServiceNetworks() ServiceNetworks {
	var serviceNetworks ServiceNetworks
	for _, serviceNetwork := range nc.serviceNetworks {
		serviceNetworks = append(serviceNetworks, serviceNetwork)
	}
	sort.Sort(serviceNetworks)
	return serviceNetworks
}

func (nc *NetworkCache) OnNewNode(k8snode *corev1.Node) {
	if _, ok := nc.nodeNetworks[k8snode.Name]; ok {
		return
	}

	var ip string
	for _, addr := range k8snode.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP || addr.Type == corev1.NodeExternalIP {
			if ip == "" {
				ip = addr.Address
			}
		}
	}

	nn := &NodeNetwork{
		Name: k8snode.Name,
		IP:   ip,
	}
	nn.SetID(GenUUID())
	nc.nodeNetworks[k8snode.Name] = nn

	if k8snode.Spec.PodCIDR != "" {
		nc.newPodNetworks(k8snode)
	}
}

func (nc *NetworkCache) newPodNetworks(k8snode *corev1.Node) {
	pn := &PodNetwork{
		NodeName: k8snode.Name,
		PodCIDR:  k8snode.Spec.PodCIDR,
		PodIPs:   make([]PodIP, 0),
	}
	pn.SetID(GenUUID())
	nc.podNetworks[k8snode.Name] = pn
}

func (nc *NetworkCache) OnNewPod(k8spod *corev1.Pod) {
	if k8spod.Status.PodIP == "" || k8spod.Status.Phase != corev1.PodRunning {
		return
	}

	podNetwork, ok := nc.podNetworks[k8spod.Spec.NodeName]
	if ok == false {
		return
	}

	if k8spod.Spec.HostNetwork == false {
		podNetwork.PodIPs = append(podNetwork.PodIPs, PodIP{
			Namespace: k8spod.Namespace,
			Name:      k8spod.Name,
			IP:        k8spod.Status.PodIP,
		})
	}
}

func (nc *NetworkCache) OnNewService(k8ssvc *corev1.Service) {
	sn := &ServiceNetwork{
		Namespace: k8ssvc.Namespace,
		Name:      k8ssvc.Name,
		IP:        k8ssvc.Spec.ClusterIP,
	}
	sn.SetID(GenUUID())
	nc.serviceNetworks[genServiceKey(k8ssvc)] = sn
}

func genServiceKey(k8ssvc *corev1.Service) string {
	return k8ssvc.Namespace + "/" + k8ssvc.Name
}

func (nc *NetworkCache) OnDeleteNode(k8snode *corev1.Node) {
	delete(nc.nodeNetworks, k8snode.Name)
	delete(nc.podNetworks, k8snode.Name)
}

func (nc *NetworkCache) OnDeletePod(k8spod *corev1.Pod) {
	if podNetwork, ok := nc.podNetworks[k8spod.Spec.NodeName]; ok {
		for i, podIP := range podNetwork.PodIPs {
			if podIP.Namespace == k8spod.Namespace && podIP.Name == k8spod.Name {
				podNetwork.PodIPs = append(podNetwork.PodIPs[:i], podNetwork.PodIPs[i+1:]...)
				break
			}
		}
	}
}

func (nc *NetworkCache) OnDeleteService(k8ssvc *corev1.Service) {
	delete(nc.serviceNetworks, genServiceKey(k8ssvc))
}

func (nc *NetworkCache) OnUpdateNode(k8snode *corev1.Node) {
	if _, ok := nc.nodeNetworks[k8snode.Name]; ok == false {
		return
	}

	if pn, ok := nc.podNetworks[k8snode.Name]; ok {
		pn.PodCIDR = k8snode.Spec.PodCIDR
		return
	}

	nc.newPodNetworks(k8snode)
}

func (nc *NetworkCache) OnUpdateService(k8ssvc *corev1.Service) {
	nc.OnNewService(k8ssvc)
}

func (nc *NetworkCache) OnUpdatePod(k8spodOld, k8spodNew *corev1.Pod) {
	if k8spodOld.Status.PodIP == k8spodNew.Status.PodIP {
		if k8spodNew.Status.Phase == corev1.PodSucceeded || k8spodNew.Status.Phase == corev1.PodFailed {
			nc.OnDeletePod(k8spodNew)
		}
		return
	}

	podNetwork, ok := nc.podNetworks[k8spodNew.Spec.NodeName]
	if ok == false {
		return
	}

	podIP := PodIP{
		Namespace: k8spodNew.Namespace,
		Name:      k8spodNew.Name,
		IP:        k8spodNew.Status.PodIP,
	}
	for i, p := range podNetwork.PodIPs {
		if p.Namespace == k8spodNew.Namespace && p.Name == k8spodNew.Name {
			podNetwork.PodIPs[i] = podIP
			return
		}
	}
	podNetwork.PodIPs = append(podNetwork.PodIPs, podIP)
}

package network

import (
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

func (nc *NetworkCache) GetNodeNetworks() []NodeNetwork {
	var nodeNetworks []NodeNetwork
	for _, nodeNetwork := range nc.nodeNetworks {
		nodeNetworks = append(nodeNetworks, *nodeNetwork)
	}
	return nodeNetworks
}

func (nc *NetworkCache) GetPodNetworks() []PodNetwork {
	var podNetworks []PodNetwork
	for _, podNetwork := range nc.podNetworks {
		podNetworks = append(podNetworks, *podNetwork)
	}
	return podNetworks
}

func (nc *NetworkCache) GetServiceNetworks() []ServiceNetwork {
	var serviceNetworks []ServiceNetwork
	for _, serviceNetwork := range nc.serviceNetworks {
		serviceNetworks = append(serviceNetworks, *serviceNetwork)
	}
	return serviceNetworks
}

func (nc *NetworkCache) OnNewNode(k8snode *corev1.Node) {
	var ip string
	for _, addr := range k8snode.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP || addr.Type == corev1.NodeExternalIP {
			if ip == "" {
				ip = addr.Address
			}
		}
	}

	nc.nodeNetworks[k8snode.Name] = &NodeNetwork{
		Name: k8snode.Name,
		IP:   ip,
	}
	nc.podNetworks[k8snode.Name] = &PodNetwork{
		NodeName: k8snode.Name,
		PodCIDR:  k8snode.Spec.PodCIDR,
	}
}

func (nc *NetworkCache) OnNewPod(k8spod *corev1.Pod) {
	podNetwork, ok := nc.podNetworks[k8spod.Spec.NodeName]
	if ok == false {
		return
	}

	podNetwork.PodIPs = append(podNetwork.PodIPs, PodIP{
		Namespace: k8spod.Namespace,
		Name:      k8spod.Name,
		IP:        k8spod.Status.PodIP,
	})
}

func (nc *NetworkCache) OnNewService(k8ssvc *corev1.Service) {
	nc.serviceNetworks[genServiceKey(k8ssvc)] = &ServiceNetwork{
		Namespace: k8ssvc.Namespace,
		Name:      k8ssvc.Name,
		IP:        k8ssvc.Spec.ClusterIP,
	}
}

func genServiceKey(k8ssvc *corev1.Service) string {
	return k8ssvc.Namespace + "/" + k8ssvc.Name
}

func (nc *NetworkCache) OnDeleteNode(k8snode *corev1.Node) {
	delete(nc.nodeNetworks, k8snode.Name)
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

func (nc *NetworkCache) OnUpdateService(k8ssvc *corev1.Service) {
	nc.OnNewService(k8ssvc)
}

func (nc *NetworkCache) OnUpdatePod(k8spod *corev1.Pod) {
	podNetwork, ok := nc.podNetworks[k8spod.Spec.NodeName]
	if ok == false {
		return
	}

	podIP := PodIP{
		Namespace: k8spod.Namespace,
		Name:      k8spod.Name,
		IP:        k8spod.Status.PodIP,
	}
	for i, p := range podNetwork.PodIPs {
		if p.Namespace == k8spod.Namespace && p.Name == k8spod.Name {
			podNetwork.PodIPs[i] = podIP
			return
		}
	}
	podNetwork.PodIPs = append(podNetwork.PodIPs, podIP)
}

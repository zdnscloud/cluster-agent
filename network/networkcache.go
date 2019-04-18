package network

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/zdnscloud/gok8s/cache"
)

type NetworkCache struct {
	nodes    map[string]*Node
	pods     map[string]*Pod
	services map[string]*Service

	cache cache.Cache
}

func newNetworkCache(cache cache.Cache) *NetworkCache {
	return &NetworkCache{
		cache:    cache,
		nodes:    make(map[string]*Node),
		pods:     make(map[string]*Pod),
		services: make(map[string]*Service),
	}
}

func (nc *NetworkCache) GetNodes() []Node {
	var nodes []Node
	for _, node := range nc.nodes {
		nodes = append(nodes, *node)
	}
	return nodes
}

func (nc *NetworkCache) GetPods() []Pod {
	var pods []Pod
	for _, pod := range nc.pods {
		pods = append(pods, *pod)
	}
	return pods
}

func (nc *NetworkCache) GetServices() []Service {
	var services []Service
	for _, service := range nc.services {
		services = append(services, *service)
	}
	return services
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

	nc.nodes[k8snode.Name] = &Node{
		Name: k8snode.Name,
		IP:   ip,
	}
	nc.pods[k8snode.Name] = &Pod{
		NodeName: k8snode.Name,
		PodCIDR:  k8snode.Spec.PodCIDR,
	}
}

func (nc *NetworkCache) OnNewPod(k8spod *corev1.Pod) {
	pod, ok := nc.pods[k8spod.Spec.NodeName]
	if ok == false {
		return
	}

	pod.PodIPs = append(pod.PodIPs, PodIP{
		Namespace: k8spod.Namespace,
		Name:      k8spod.Name,
		IP:        k8spod.Status.PodIP,
	})
}

func (nc *NetworkCache) OnNewService(k8ssvc *corev1.Service) {
	nc.services[genServiceKey(k8ssvc)] = &Service{
		Namespace: k8ssvc.Namespace,
		Name:      k8ssvc.Name,
		IP:        k8ssvc.Spec.ClusterIP,
	}
}

func genServiceKey(k8ssvc *corev1.Service) string {
	return k8ssvc.Namespace + "/" + k8ssvc.Name
}

func (nc *NetworkCache) OnDeleteNode(k8snode *corev1.Node) {
	delete(nc.nodes, k8snode.Name)
}

func (nc *NetworkCache) OnDeletePod(k8spod *corev1.Pod) {
	if pod, ok := nc.pods[k8spod.Spec.NodeName]; ok {
		for i, podIP := range pod.PodIPs {
			if podIP.Namespace == k8spod.Namespace && podIP.Name == k8spod.Name {
				pod.PodIPs = append(pod.PodIPs[:i], pod.PodIPs[i+1:]...)
				break
			}
		}
	}
}

func (nc *NetworkCache) OnDeleteService(k8ssvc *corev1.Service) {
	delete(nc.services, genServiceKey(k8ssvc))
}

func (nc *NetworkCache) OnUpdateService(k8ssvc *corev1.Service) {
	nc.OnNewService(k8ssvc)
}

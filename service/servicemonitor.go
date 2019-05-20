package service

import (
	"context"
	"fmt"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/helper"
)

const (
	AnnkeyForUDPIngress = "zcloud_ingress_udp"
	RunningState        = "Running"
)

type Service struct {
	Name      string
	Ingress   StringSet
	Workloads []*Workload
}

type Ingress struct {
	Name  string
	Rules []IngressRule
}

type IngressRule struct {
	Host     string          `json:"host"`
	Port     int             `json:"port,omitempty"`
	Protocol IngressProtocol `json:"protocol"`
	Paths    []IngressPath   `json:"paths"`
}

type IngressPath struct {
	Path        string
	ServiceName string
	ServicePort int
}

type ServiceMonitor struct {
	services map[string]*Service
	ings     map[string]*Ingress
	lock     sync.RWMutex

	cache cache.Cache
}

func newServiceMonitor(cache cache.Cache) *ServiceMonitor {
	return &ServiceMonitor{
		cache:    cache,
		services: make(map[string]*Service),
		ings:     make(map[string]*Ingress),
	}
}

func (s *ServiceMonitor) GetInnerServices() []InnerService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	svcs := make([]InnerService, 0, len(s.services))
	for _, svc := range s.services {
		if len(svc.Ingress) == 0 {
			svcs = append(svcs, InnerService{
				Name:      svc.Name,
				Workloads: svc.Workloads,
			})
		}
	}
	return svcs
}

func (s *ServiceMonitor) GetOuterServices() []OuterService {
	s.lock.RLock()
	defer s.lock.RUnlock()
	outerSvcs := make([]OuterService, 0, len(s.services))
	//handle several services shared same ingress
	returnedIngress := NewStringSet()
	for _, svc := range s.services {
		for ing := range svc.Ingress {
			if returnedIngress.Member(ing) == false {
				outerSvcs = append(outerSvcs, s.toOuterService(s.ings[ing])...)
				returnedIngress.Add(ing)
			}
		}
	}
	return outerSvcs
}

func (s *ServiceMonitor) toOuterService(ing *Ingress) []OuterService {
	outerSvcs := make([]OuterService, 0, len(ing.Rules))
	var outerSvc OuterService
	for _, rule := range ing.Rules {
		if rule.Protocol == IngressProtocolHTTP {
			outerSvc.EntryPoint = fmt.Sprintf("%s://%s", rule.Protocol, rule.Host)
		} else {
			outerSvc.EntryPoint = fmt.Sprintf("%s:%d", rule.Protocol, rule.Port)
		}
		innerSvcs := make(map[string]InnerService)
		for _, p := range rule.Paths {
			svc, ok := s.services[p.ServiceName]
			if ok {
				innerSvcs[p.Path] = InnerService{
					Name:      svc.Name,
					Workloads: svc.Workloads,
				}
			}
		}
		outerSvc.Services = innerSvcs
		outerSvcs = append(outerSvcs, outerSvc)
	}
	return outerSvcs
}

func (s *ServiceMonitor) OnNewService(k8ssvc *corev1.Service) {
	svc, err := s.k8ssvcToSCService(k8ssvc)
	if err != nil {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.services[svc.Name] = svc
	for name, ing := range s.ings {
		ss := ingressLinkedServices(ing)
		if ss.Member(svc.Name) {
			s.linkIngressToService(name, svc.Name)
		}
	}
}

func (s *ServiceMonitor) k8ssvcToSCService(k8ssvc *corev1.Service) (*Service, error) {
	svc := &Service{
		Name: k8ssvc.Name,
	}

	if len(k8ssvc.Spec.Selector) == 0 {
		return svc, nil
	}

	ls := metav1.LabelSelector{
		MatchLabels: k8ssvc.Spec.Selector,
	}
	k8spods := corev1.PodList{}
	opts := &client.ListOptions{Namespace: k8ssvc.Namespace}
	labels, _ := metav1.LabelSelectorAsSelector(&ls)
	opts.LabelSelector = labels
	err := s.cache.List(context.TODO(), opts, &k8spods)
	if err != nil {
		log.Warnf("get pod list failed:%s", err.Error())
		return nil, err
	}

	workerLoads := make(map[string]*Workload)
	for _, k8spod := range k8spods.Items {
		pod := Pod{
			Name:  k8spod.Name,
			State: helper.GetPodState(&k8spod),
		}

		if len(k8spod.OwnerReferences) == 1 {
			name, kind, succeed := s.getPodOwner(k8spod.Namespace, k8spod.OwnerReferences[0])
			if succeed == false {
				continue
			}

			wl, ok := workerLoads[name]
			if ok == false {
				wl = &Workload{
					Name: name,
					Kind: kind,
				}
			}
			wl.Pods = append(wl.Pods, pod)
			workerLoads[name] = wl
		}
	}

	for _, wl := range workerLoads {
		svc.Workloads = append(svc.Workloads, wl)
	}
	svc.Ingress = NewStringSet()
	return svc, nil
}

func (s *ServiceMonitor) getPodOwner(namespace string, owner metav1.OwnerReference) (string, string, bool) {
	if owner.Kind != "ReplicaSet" {
		return owner.Name, owner.Kind, true
	}

	var k8srs appsv1.ReplicaSet
	err := s.cache.Get(context.TODO(), k8stypes.NamespacedName{namespace, owner.Name}, &k8srs)
	if err != nil {
		log.Warnf("get replicaset failed:%s", err.Error())
		return "", "", false
	}

	if len(k8srs.OwnerReferences) != 1 {
		log.Warnf("replicaset OwnerReferences is strange:%v", k8srs.OwnerReferences)
		return "", "", false
	}

	owner = k8srs.OwnerReferences[0]
	if owner.Kind != "Deployment" {
		log.Warnf("replicaset parent is not deployment but %v", owner.Kind)
		return "", "", false
	}
	return owner.Name, owner.Kind, true
}

func (s *ServiceMonitor) OnDeleteService(svc *corev1.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.services, svc.Name)
}

func (s *ServiceMonitor) OnUpdateService(old, new *corev1.Service) {
	if isMapEqual(old.Spec.Selector, new.Spec.Selector) {
		return
	}
	s.OnNewService(new)
}

func (s *ServiceMonitor) OnUpdatePod(old, new *corev1.Pod) {
	oldState := helper.GetPodState(old)
	newState := helper.GetPodState(new)
	if newState == oldState {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	for _, s := range s.services {
		for _, wl := range s.Workloads {
			for i, pod := range wl.Pods {
				if pod.Name == new.Name {
					wl.Pods[i].State = newState
					return
				}
			}
		}
	}
}

func (s *ServiceMonitor) OnUpdateEndpoints(old, new *corev1.Endpoints) {
	if len(old.Subsets) == 0 && len(new.Subsets) == 0 {
		return
	}

	s.lock.Lock()
	hasPodChange := s.hasPodNameChange(new)
	s.lock.Unlock()

	if hasPodChange {
		var k8ssvc corev1.Service
		err := s.cache.Get(context.TODO(), k8stypes.NamespacedName{new.Namespace, new.Name}, &k8ssvc)
		if err != nil {
			log.Warnf("get service %s failed:%s", new.Name, err.Error())
			return
		}
		s.OnNewService(&k8ssvc)
	}
}

func (s *ServiceMonitor) hasPodNameChange(eps *corev1.Endpoints) bool {
	svc, ok := s.services[eps.Name]
	if ok == false {
		log.Warnf("endpoints %s has no related service", eps.Name)
		return false
	}

	pods := make(map[string]Pod)
	for _, wl := range svc.Workloads {
		for _, p := range wl.Pods {
			pods[p.Name] = p
		}
	}

	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			if addr.TargetRef != nil {
				n := addr.TargetRef.Name
				if p, ok := pods[n]; ok == false {
					return true
				} else {
					p.State = RunningState
				}
			}
		}

		for _, addr := range subset.NotReadyAddresses {
			if addr.TargetRef != nil {
				n := addr.TargetRef.Name
				if p, ok := pods[n]; ok == false {
					return true
				} else {
					if p.State == RunningState {
						var pod corev1.Pod
						err := s.cache.Get(context.TODO(), k8stypes.NamespacedName{eps.Namespace, n}, &pod)
						if err != nil {
							log.Warnf("get pod %s in namespace %s failed", n, eps.Namespace)
						} else {
							p.State = helper.GetPodState(&pod)
						}
					}
				}
			}
		}
	}
	return false
}

func (s *ServiceMonitor) OnNewIngress(k8sing *extv1beta1.Ingress) {
	ing := k8sIngressToSCIngress(k8sing)
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addIngress(ing)
}

func (s *ServiceMonitor) addIngress(ing *Ingress) {
	old, ok := s.ings[ing.Name]
	involedServices := ingressLinkedServices(ing)
	if ok {
		oldServices := ingressLinkedServices(old)
		old.Rules = append(old.Rules, ing.Rules...)
		involedServices = involedServices.Difference(oldServices)
	} else {
		s.ings[ing.Name] = ing
	}

	for service := range involedServices {
		s.linkIngressToService(ing.Name, service)
	}
}

func (s *ServiceMonitor) OnNewTransportLayerIngress(ing *Ingress) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addIngress(ing)
}

func (s *ServiceMonitor) OnReplaceTransportLayerIngress(old, new *Ingress) {
	s.lock.Lock()
	defer s.lock.Unlock()
}

func (s *ServiceMonitor) linkIngressToService(ing, service string) {
	svc, ok := s.services[service]
	if ok == false {
		log.Warnf("unknown service %s specified in ingress %s", service, ing)
	} else {
		svc.Ingress.Add(ing)
	}
}

func (s *ServiceMonitor) removeIngressFromService(ing, service string) {
	svc, ok := s.services[service]
	if ok == false {
		log.Warnf("unknown service %s specified in ingress %s", service, ing)
	} else {
		svc.Ingress.Delete(ing)
	}
}

func k8sIngressToSCIngress(k8sing *extv1beta1.Ingress) *Ingress {
	ing := &Ingress{
		Name: k8sing.Name,
	}
	k8srules := k8sing.Spec.Rules
	var rules []IngressRule
	for _, rule := range k8srules {
		http := rule.HTTP
		if http == nil {
			continue
		}

		var paths []IngressPath
		for _, p := range http.Paths {
			paths = append(paths, IngressPath{
				ServiceName: p.Backend.ServiceName,
				Path:        p.Path,
			})
		}

		rules = append(rules, IngressRule{
			Host:     rule.Host,
			Paths:    paths,
			Protocol: IngressProtocolHTTP,
		})
	}

	ing.Rules = rules
	return ing
}

func (s *ServiceMonitor) OnUpdateIngress(old, new *extv1beta1.Ingress) {
	oldIng := k8sIngressToSCIngress(old)
	newIng := k8sIngressToSCIngress(new)

	s.lock.Lock()
	defer s.lock.Unlock()

	s.ings[new.Name] = newIng
	oldServices := ingressLinkedServices(oldIng)
	newServices := ingressLinkedServices(newIng)
	for name := range oldServices.Difference(newServices) {
		s.removeIngressFromService(oldIng.Name, name)
	}

	for name := range newServices.Difference(oldServices) {
		s.linkIngressToService(newIng.Name, name)
	}
}

func (s *ServiceMonitor) updateIngress(old, new *Ingress) {
	old_, ok := s.ings[old.Name]
	if ok == false {
		log.Errorf("update unknown ingress %s", old.Name)
		return
	}

	if len(old.Rules) == 0 {
		log.Errorf("update ingress with empty rule %s", old.Name)
		return
	}

	oldServices := ingressLinkedServices(old_)
	ingressRemoveRules(old_, old.Rules[0].Protocol)
	if new != nil {
		old_.Rules = append(old_.Rules, new.Rules...)
	}
	newServices := ingressLinkedServices(old_)
	for name := range oldServices.Difference(newServices) {
		s.removeIngressFromService(old.Name, name)
	}
	for name := range newServices.Difference(oldServices) {
		s.linkIngressToService(old.Name, name)
	}

	if len(old_.Rules) == 0 {
		delete(s.ings, old_.Name)
	}
}

func (s *ServiceMonitor) OnDeleteIngress(k8sing *extv1beta1.Ingress) {
	ing := k8sIngressToSCIngress(k8sing)
	s.lock.Lock()
	defer s.lock.Unlock()

	s.updateIngress(ing, nil)
}

func (s *ServiceMonitor) OnDeleteTransportLayerIngress(ing *Ingress) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.updateIngress(ing, nil)
}

func isMapEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v1 := range m1 {
		v2, ok := m2[k]
		if ok == false || v1 != v2 {
			return false
		}
	}
	return true
}

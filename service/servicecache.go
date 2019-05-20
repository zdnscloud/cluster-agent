package service

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
)

type ServiceCache struct {
	services map[string]*ServiceMonitor
	lock     sync.RWMutex
	cache    cache.Cache
	stopCh   chan struct{}
}

func NewServiceCache(c cache.Cache) (*ServiceCache, error) {
	ctrl := controller.New("serviceCache", c, scheme.Scheme)
	ctrl.Watch(&corev1.Namespace{})
	ctrl.Watch(&corev1.Service{})
	ctrl.Watch(&corev1.Endpoints{})
	ctrl.Watch(&corev1.Pod{})
	ctrl.Watch(&corev1.ConfigMap{})
	ctrl.Watch(&extv1beta1.Ingress{})

	stopCh := make(chan struct{})
	sc := &ServiceCache{
		stopCh: stopCh,
		cache:  c,
	}
	if err := sc.initServices(); err != nil {
		return nil, err
	}

	go ctrl.Start(stopCh, sc, predicate.NewIgnoreUnchangedUpdate())
	return sc, nil
}

func (r *ServiceCache) initServices() error {
	nses := &corev1.NamespaceList{}
	err := r.cache.List(context.TODO(), nil, nses)
	if err != nil {
		return err
	}

	services := make(map[string]*ServiceMonitor)
	for _, ns := range nses.Items {
		s := newServiceMonitor(r.cache)
		services[ns.Name] = s
	}
	r.services = services
	return nil
}

func (r *ServiceCache) GetInnerServices(namespace string) []InnerService {
	r.lock.RLock()
	monitor, ok := r.services[namespace]
	r.lock.RUnlock()

	if ok == false {
		return nil
	}
	return monitor.GetInnerServices()
}

func (r *ServiceCache) GetOuterServices(namespace string) []OuterService {
	r.lock.RLock()
	monitor, ok := r.services[namespace]
	r.lock.RUnlock()
	if ok == false {
		return nil
	}
	return monitor.GetOuterServices()
}

func (r *ServiceCache) OnCreate(e event.CreateEvent) (handler.Result, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		if _, ok := r.services[obj.Name]; ok == false {
			s := newServiceMonitor(r.cache)
			r.services[obj.Name] = s
		}
	case *corev1.Service:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnNewService(obj)
		}
	case *corev1.ConfigMap:
		r.onNewTransportLayerIngress(obj)
	case *extv1beta1.Ingress:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnNewIngress(obj)
		}
	}

	return handler.Result{}, nil
}

func (r *ServiceCache) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch newObj := e.ObjectNew.(type) {
	case *corev1.Service:
		s, ok := r.services[newObj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", newObj.Namespace)
		} else {
			s.OnUpdateService(e.ObjectOld.(*corev1.Service), newObj)
		}
	case *corev1.Pod:
		s, ok := r.services[newObj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", newObj.Namespace)
		} else {
			s.OnUpdatePod(e.ObjectOld.(*corev1.Pod), newObj)
		}
	case *corev1.Endpoints:
		s, ok := r.services[newObj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", newObj.Namespace)
		} else {
			s.OnUpdateEndpoints(e.ObjectOld.(*corev1.Endpoints), newObj)
		}
	case *corev1.ConfigMap:
		r.onUpdateTransportLayerIngress(e.ObjectOld.(*corev1.ConfigMap), newObj)
	case *extv1beta1.Ingress:
		s, ok := r.services[newObj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", newObj.Namespace)
		} else {
			s.OnUpdateIngress(e.ObjectOld.(*extv1beta1.Ingress), newObj)
		}
	}

	return handler.Result{}, nil
}

func (r *ServiceCache) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	switch obj := e.Object.(type) {
	case *corev1.Namespace:
		_, ok := r.services[obj.Name]
		if ok == false {
			log.Warnf("namespace %s isn't included in repo", obj.Name)
		} else {
			delete(r.services, obj.Name)
		}
	case *corev1.Service:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnDeleteService(obj)
		}
	case *extv1beta1.Ingress:
		s, ok := r.services[obj.Namespace]
		if ok == false {
			log.Errorf("namespace %s is unknown", obj.Namespace)
		} else {
			s.OnDeleteIngress(obj)
		}
	}

	return handler.Result{}, nil
}

func (r *ServiceCache) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (r *ServiceCache) onNewTransportLayerIngress(cm *corev1.ConfigMap) {
	if cm.Namespace == NginxIngressNamespace &&
		(cm.Name == NginxUDPConfigMapName || cm.Name == NginxTCPConfigMapName) {
		protocol := protocolForConfigMap(cm.Name)
		namespaceAndIngs, err := configMapToIngresses(cm.Data, protocol)
		if err != nil {
			log.Errorf("invalid configmap:%s", err.Error())
			return
		}

		for namespace, ings := range namespaceAndIngs {
			s, ok := r.services[namespace]
			if ok == false {
				log.Errorf("namespace %s is unknown", namespace)
			} else {
				for _, ing := range ings {
					s.OnNewTransportLayerIngress(ing)
				}
			}
		}
	}
}

func (r *ServiceCache) onUpdateTransportLayerIngress(old, new *corev1.ConfigMap) {
	if new.Namespace == NginxIngressNamespace &&
		(new.Name == NginxUDPConfigMapName || new.Name == NginxTCPConfigMapName) {
		protocol := protocolForConfigMap(new.Name)

		oldNamespaceAndIngs, err := configMapToIngresses(old.Data, protocol)
		if err != nil {
			log.Errorf("invalid transport ingress config %s with err %s", old.Name, err.Error())
			return
		}

		newNamespaceAndIngs, err := configMapToIngresses(new.Data, protocol)
		if err != nil {
			log.Errorf("invalid transport ingress config %s with err %s", new.Name, err.Error())
			return
		}

		for namespace, newIngs := range newNamespaceAndIngs {
			s, ok := r.services[namespace]
			if ok == false {
				log.Errorf("namespace %s is unknown", namespace)
				continue
			}

			oldIngs, ok := oldNamespaceAndIngs[namespace]
			if ok == false {
				for _, ing := range newIngs {
					s.OnNewTransportLayerIngress(ing)
				}
			} else {
				for name, ing := range newIngs {
					if old, ok := oldIngs[name]; ok {
						delete(oldIngs, name)
						s.OnReplaceTransportLayerIngress(old, ing)
					} else {
						s.OnNewTransportLayerIngress(ing)
					}
				}
			}
		}

		for namespace, oldIngs := range oldNamespaceAndIngs {
			s, ok := r.services[namespace]
			if ok == false {
				log.Errorf("namespace %s is unknown", namespace)
				continue
			}
			for _, ing := range oldIngs {
				s.OnDeleteTransportLayerIngress(ing)
			}
		}
	}
}

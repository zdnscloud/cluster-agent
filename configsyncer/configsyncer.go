package configsyncer

import (
	"context"
	"fmt"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
)

type ConfigSyncer struct {
	lock        sync.RWMutex
	client      client.Client
	stopCh      chan struct{}
	configOwner *ConfigOwner
}

func NewConfigSyncer(cli client.Client, c cache.Cache) *ConfigSyncer {
	ctrl := controller.New("configSyncer", c, scheme.Scheme)
	ctrl.Watch(&appsv1.Deployment{})
	ctrl.Watch(&appsv1.StatefulSet{})
	ctrl.Watch(&appsv1.DaemonSet{})
	ctrl.Watch(&corev1.ConfigMap{})
	ctrl.Watch(&corev1.Secret{})
	stopCh := make(chan struct{})
	syncer := &ConfigSyncer{
		stopCh:      stopCh,
		client:      cli,
		configOwner: newConfigOwner(),
	}
	go ctrl.Start(stopCh, syncer, predicate.NewIgnoreUnchangedUpdate())
	return syncer
}

func (syncer *ConfigSyncer) OnCreate(e event.CreateEvent) (handler.Result, error) {
	syncer.lock.Lock()
	defer syncer.lock.Unlock()

	var pc PodController
	switch obj := e.Object.(type) {
	case *appsv1.Deployment:
		pc = &deployment{obj}
	case *appsv1.StatefulSet:
		pc = &statefulset{obj}
	case *appsv1.DaemonSet:
		pc = &daemonset{obj}
	}

	if pc != nil {
		if hasRequiredAnnotation(pc) {
			syncer.configOwner.OnNewPodController(pc)
		}
	}

	return handler.Result{}, nil
}

func (syncer *ConfigSyncer) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	var pcKey string
	var foundPc bool
	switch newObj := e.ObjectNew.(type) {
	case *corev1.ConfigMap:
		pcKey, foundPc = syncer.configOwner.GetPodControllerUseConfig(newObj.Namespace, ObjectKey(newObj))
	case *corev1.Secret:
		pcKey, foundPc = syncer.configOwner.GetPodControllerUseConfig(newObj.Namespace, ObjectKey(newObj))
	}

	if foundPc {
		pc, err := syncer.getPodController(e.MetaNew.GetNamespace(), pcKey)
		if err != nil {
			log.Errorf("get workerload failed:%s", err.Error())
		} else {
			hash := getConfigHash(pc)
			newHash, _ := syncer.calculatePodControllerConfigHash(pc)
			if hash != newHash {
				setConfigHash(pc, newHash)
				if err := syncer.updatePodController(pc); err != nil {
					log.Errorf("update pc %v failed %v", pc, err.Error())
				}
			}
		}
	}

	return handler.Result{}, nil
}

func (syncer *ConfigSyncer) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (syncer *ConfigSyncer) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (syncer *ConfigSyncer) updatePodController(pc PodController) error {
	return syncer.client.Update(context.TODO(), pc.GetObject())
}

func (syncer *ConfigSyncer) getPodController(namespace, pcKey string) (PodController, error) {
	kind, name := ParseKey(pcKey)
	var obj runtime.Object
	var pc PodController
	switch kind {
	case KindDeployment:
		var deploy appsv1.Deployment
		obj = &deploy
		pc = &deployment{&deploy}
	case KindStatefulSet:
		var statefulSet appsv1.StatefulSet
		obj = &statefulSet
		pc = &statefulset{&statefulSet}
	case KindDaemonSet:
		var daemonSet appsv1.DaemonSet
		obj = &daemonSet
		pc = &daemonset{&daemonSet}
	default:
		return nil, fmt.Errorf("unsupported pod controller with kind:%s", kind)
	}

	err := syncer.client.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, obj)
	if err != nil {
		return nil, err
	}
	return pc, nil
}

func (syncer *ConfigSyncer) getConfig(namespace, objKey string) (runtime.Object, error) {
	kind, name := ParseKey(objKey)
	var obj runtime.Object
	switch kind {
	case KindConfigMap:
		var configMap corev1.ConfigMap
		obj = &configMap
	case KindSecret:
		var secret corev1.Secret
		obj = &secret
	default:
		return nil, fmt.Errorf("unsupported config kind:%s", kind)
	}

	err := syncer.client.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, obj)
	if err != nil {
		return nil, err
	} else {
		return obj, nil
	}
}

func (syncer *ConfigSyncer) calculatePodControllerConfigHash(obj PodController) (string, error) {
	configs := getReferedConfig(obj)
	objects := make([]runtime.Object, 0, len(configs))
	for _, config := range configs {
		if obj, err := syncer.getConfig(obj.GetNamespace(), config); err != nil {
			return "", err
		} else {
			objects = append(objects, obj)
		}
	}
	return calculateConfigHash(objects)
}

func hasRequiredAnnotation(obj PodController) bool {
	annotations := obj.GetAnnotations()
	if value, ok := annotations[RequiredAnnotation]; ok {
		if value == requiredAnnotationValue {
			return true
		}
	}
	return false
}

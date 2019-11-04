package configsyncer

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/helper"
	"github.com/zdnscloud/gok8s/predicate"
)

const (
	ZcloudFinalizer = "zcloud.cn/finalizer"
)

type ConfigSyncer struct {
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

func (syncer *ConfigSyncer) OnCreate(e event.CreateEvent) (result handler.Result, err error) {
	var pc PodController
	switch obj := e.Object.(type) {
	case *appsv1.Deployment:
		pc = &deployment{obj}
	case *appsv1.StatefulSet:
		pc = &statefulset{obj}
	case *appsv1.DaemonSet:
		pc = &daemonset{obj}
	case *corev1.ConfigMap:
		syncer.onNewConfig(obj)
	case *corev1.Secret:
		syncer.onNewConfig(obj)
	}

	if pc != nil && hasRequiredAnnotation(pc) {
		syncer.onNewPodController(pc)
	}

	return handler.Result{}, nil
}

func (syncer *ConfigSyncer) onNewConfig(config Object) {
	if helper.HasFinalizer(config, ZcloudFinalizer) {
		return
	}

	pcKeys := syncer.configOwner.GetPodControllersUseConfig(config.GetNamespace(), ObjectKey(config))
	if len(pcKeys) != 0 {
		helper.AddFinalizer(config, ZcloudFinalizer)
		if err := syncer.client.Update(context.TODO(), config); err != nil {
			log.Errorf("add finalizer to %s failed %s", config.GetName(), err.Error())
		}
	}
}

func (syncer *ConfigSyncer) onNewPodController(pc PodController) {
	configs := getReferedConfig(pc)
	if len(configs) == 0 {
		return
	}

	namespace := pc.GetNamespace()
	for _, configKey := range configs {
		config, err := syncer.getConfig(namespace, configKey)
		if err != nil {
			log.Errorf("get %s failed %s", configKey, err.Error())
			continue
		}
		metaObj := config.(metav1.Object)
		if helper.HasFinalizer(metaObj, ZcloudFinalizer) {
			continue
		}
		helper.AddFinalizer(metaObj, ZcloudFinalizer)
		if err := syncer.client.Update(context.TODO(), config); err != nil {
			log.Errorf("add finalizer to %s failed %s", configKey, err.Error())
			return
		}
	}
	syncer.configOwner.OnNewPodController(pc, configs)
}

func (syncer *ConfigSyncer) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	var oldConfig, newConfig Object
	var oldPc, newPc PodController
	switch newObj := e.ObjectNew.(type) {
	case *corev1.ConfigMap:
		oldConfig = e.ObjectOld.(*corev1.ConfigMap)
		newConfig = newObj
	case *corev1.Secret:
		oldConfig = e.ObjectOld.(*corev1.Secret)
		newConfig = newObj
	case *appsv1.Deployment:
		oldPc = &deployment{e.ObjectOld.(*appsv1.Deployment)}
		newPc = &deployment{newObj}
	case *appsv1.StatefulSet:
		oldPc = &statefulset{e.ObjectOld.(*appsv1.StatefulSet)}
		newPc = &statefulset{newObj}
	case *appsv1.DaemonSet:
		oldPc = &daemonset{e.ObjectOld.(*appsv1.DaemonSet)}
		newPc = &daemonset{newObj}
	}

	if oldPc != nil && newPc != nil && hasRequiredAnnotation(newPc) {
		syncer.configOwner.OnUpdatePodController(oldPc, newPc)
	} else if oldConfig != nil && newConfig != nil {
		syncer.onConfigChange(oldConfig, newConfig)
	}

	return handler.Result{}, nil
}

func (syncer *ConfigSyncer) onConfigChange(oldConfig, newConfig Object) {
	//handle configure delete
	if oldConfig.GetDeletionTimestamp() == nil && newConfig.GetDeletionTimestamp() != nil {
		if helper.HasFinalizer(newConfig, ZcloudFinalizer) {
			pcKeys := syncer.configOwner.GetPodControllersUseConfig(newConfig.GetNamespace(), ObjectKey(newConfig))
			if len(pcKeys) == 0 {
				helper.RemoveFinalizer(newConfig, ZcloudFinalizer)
				if err := syncer.client.Update(context.TODO(), newConfig); err != nil {
					log.Errorf("update %s failed:%s", ObjectKey(newConfig), err.Error())
				}
			} else {
				log.Warnf("delete %s is still in use", ObjectKey(newConfig))
			}
		}
	} else {
		//handle configure data change
		namespace := newConfig.GetNamespace()
		pcKeys := syncer.configOwner.GetPodControllersUseConfig(namespace, ObjectKey(newConfig))
		for _, pcKey := range pcKeys {
			pc, err := syncer.getPodController(namespace, pcKey)
			if err != nil {
				log.Errorf("get workerload failed:%s", err.Error())
			} else {
				hash := getConfigHash(pc)
				newHash, _ := syncer.calculatePodControllerConfigHash(pc)
				if hash != newHash {
					setConfigHash(pc, newHash)
					if err := syncer.updatePodController(pc); err != nil {
						log.Errorf("update %s failed %v", ObjectKey(pc), err.Error())
					} else {
						log.Infof("detect workload %s configure changed, and will be restart", ObjectKey(pc))
					}
				}
			}
		}
	}

}

func (syncer *ConfigSyncer) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	var pc PodController
	switch obj := e.Object.(type) {
	case *appsv1.Deployment:
		pc = &deployment{obj}
	case *appsv1.StatefulSet:
		pc = &statefulset{obj}
	case *appsv1.DaemonSet:
		pc = &daemonset{obj}
	}

	if pc != nil && hasRequiredAnnotation(pc) {
		syncer.onDeletePodController(pc)
	}

	return handler.Result{}, nil
}

func (syncer *ConfigSyncer) onDeletePodController(pc PodController) {
	usedConfigs := getReferedConfig(pc)
	syncer.configOwner.OnDeletePodController(pc)
	namespace := pc.GetNamespace()
	for _, configKey := range usedConfigs {
		pcKeys := syncer.configOwner.GetPodControllersUseConfig(namespace, configKey)
		if len(pcKeys) > 0 {
			continue
		}

		config, err := syncer.getConfig(namespace, configKey)
		if err != nil {
			log.Errorf("get config %s failed %s", configKey, err.Error())
			continue
		}

		metaObj := config.(metav1.Object)
		if metaObj.GetDeletionTimestamp() != nil {
			helper.RemoveFinalizer(metaObj, ZcloudFinalizer)
			if err := syncer.client.Update(context.TODO(), config); err != nil {
				log.Errorf("remove finalizer of %s failed:%s", configKey, err.Error())
			} else {
				log.Infof("remove finalizer of %s since last workload used it has been removed", configKey)
			}
		}
	}
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

package configsyncer

import (
	"sync"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/set"
)

type PodControllerAndConfigs map[string][]string

type ConfigOwner struct {
	lock            sync.Mutex
	ownerAndConfigs map[string]PodControllerAndConfigs
}

func newConfigOwner() *ConfigOwner {
	return &ConfigOwner{
		ownerAndConfigs: make(map[string]PodControllerAndConfigs),
	}
}

func (owner *ConfigOwner) OnNewPodController(pc PodController) {
	configs := getReferedConfig(pc)
	if len(configs) == 0 {
		return
	}

	owner.lock.Lock()
	defer owner.lock.Unlock()

	ownerAndConfig, ok := owner.ownerAndConfigs[pc.GetNamespace()]
	if ok == false {
		ownerAndConfig = make(PodControllerAndConfigs)
		owner.ownerAndConfigs[pc.GetNamespace()] = ownerAndConfig
	}
	ownerAndConfig[ObjectKey(pc)] = configs
}

func (owner *ConfigOwner) OnUpdatePodController(oldPc, newPc PodController) {
	oldConfigs := getReferedConfig(oldPc)
	newConfigs := getReferedConfig(newPc)
	if configEq(oldConfigs, newConfigs) {
		return
	}

	owner.lock.Lock()
	defer owner.lock.Unlock()
	ownerAndConfig, ok := owner.ownerAndConfigs[newPc.GetNamespace()]
	if ok == false {
		log.Errorf("update workload %s with unknown namespace %s", ObjectKey(newPc), newPc.GetNamespace())
	} else {
		if len(newConfigs) == 0 {
			delete(ownerAndConfig, ObjectKey(newPc))
		} else {
			ownerAndConfig[ObjectKey(newPc)] = newConfigs
		}
	}
}

func (owner *ConfigOwner) OnDeletePodController(pc PodController) {
	owner.lock.Lock()
	defer owner.lock.Unlock()

	ownerAndConfig, ok := owner.ownerAndConfigs[pc.GetNamespace()]
	if ok {
		delete(ownerAndConfig, ObjectKey(pc))
	} else {
		log.Errorf("delete workload %s with unknown namespace %s", ObjectKey(pc), pc.GetNamespace())
	}
}

func (owner *ConfigOwner) GetPodControllersUseConfig(namespace, objKey string) []string {
	owner.lock.Lock()
	defer owner.lock.Unlock()

	ownerAndConfig, ok := owner.ownerAndConfigs[namespace]
	if ok == false {
		return nil
	}

	var controllers []string
	for key, configs := range ownerAndConfig {
		for _, config := range configs {
			if config == objKey {
				controllers = append(controllers, key)
			}
		}
	}
	return controllers
}

func getReferedConfig(obj PodController) []string {
	configs := set.NewStringSet()
	for _, vol := range obj.GetPodTemplate().Spec.Volumes {
		if cm := vol.VolumeSource.ConfigMap; cm != nil {
			configs.Add(GenKey(KindConfigMap, cm.Name))
		}
		if s := vol.VolumeSource.Secret; s != nil {
			configs.Add(GenKey(KindSecret, s.SecretName))
		}
	}

	for _, container := range obj.GetPodTemplate().Spec.Containers {
		for _, env := range container.EnvFrom {
			if cm := env.ConfigMapRef; cm != nil {
				configs.Add(GenKey(KindConfigMap, cm.Name))
			}
			if s := env.SecretRef; s != nil {
				configs.Add(GenKey(KindSecret, s.Name))
			}
		}

		for _, env := range container.Env {
			if valFrom := env.ValueFrom; valFrom != nil {
				if cm := valFrom.ConfigMapKeyRef; cm != nil {
					configs.Add(GenKey(KindConfigMap, cm.Name))
				}
				if s := valFrom.SecretKeyRef; s != nil {
					configs.Add(GenKey(KindSecret, s.Name))
				}
			}
		}
	}

	return configs.ToSortedSlice()
}

func configEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, s := range a {
		if b[i] != s {
			return false
		}
	}
	return true
}

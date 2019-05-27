package configsyncer

import (
	"sync"

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

	owner.lock.Lock()
	defer owner.lock.Unlock()

	ownerAndConfig, ok := owner.ownerAndConfigs[pc.GetNamespace()]
	if ok == false {
		ownerAndConfig = make(PodControllerAndConfigs)
		owner.ownerAndConfigs[pc.GetNamespace()] = ownerAndConfig
	}
	ownerAndConfig[ObjectKey(pc)] = configs
}

func (owner *ConfigOwner) GetPodControllerUseConfig(namespace, objKey string) (string, bool) {
	owner.lock.Lock()
	defer owner.lock.Unlock()

	ownerAndConfig, ok := owner.ownerAndConfigs[namespace]
	if ok == false {
		return "", false
	}

	for key, configs := range ownerAndConfig {
		for _, config := range configs {
			if config == objKey {
				return key, true
			}
		}
	}
	return "", false
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
package configsyncer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func calculateConfigHash(objects []runtime.Object) (string, error) {
	hashSource := struct {
		ConfigMaps map[string]map[string]string `json:"configMaps"`
		Secrets    map[string]map[string][]byte `json:"secrets"`
	}{
		ConfigMaps: make(map[string]map[string]string),
		Secrets:    make(map[string]map[string][]byte),
	}

	for _, obj := range objects {
		switch o := obj.(type) {
		case *corev1.ConfigMap:
			hashSource.ConfigMaps[o.Name] = o.Data
		case *corev1.Secret:
			hashSource.Secrets[o.Name] = o.Data
		default:
			return "", fmt.Errorf("unknown config type %v", obj)
		}
	}

	jsonData, err := json.Marshal(hashSource)
	if err != nil {
		return "", fmt.Errorf("unable to marshal JSON: %v", err)
	}

	hashBytes := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hashBytes), nil
}

func setConfigHash(obj PodController, hash string) {
	podTemplate := obj.GetPodTemplate()
	annotations := podTemplate.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[ConfigHashAnnotation] = hash
	podTemplate.SetAnnotations(annotations)
	obj.SetPodTemplate(podTemplate)
}

func getConfigHash(obj PodController) string {
	podTemplate := obj.GetPodTemplate()
	annotations := podTemplate.GetAnnotations()
	if annotations == nil {
		return ""
	}

	hash, ok := annotations[ConfigHashAnnotation]
	if ok {
		return hash
	} else {
		return ""
	}
}

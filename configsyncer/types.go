package configsyncer

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ConfigHashAnnotation    = "zcloud.cn/config-hash"
	FinalizerString         = "zcloud.cn/finalizer"
	RequiredAnnotation      = "zcloud.cn/update-on-config-change"
	requiredAnnotationValue = "true"

	KindConfigMap   = "ConfigMap"
	KindSecret      = "Secret"
	KindDeployment  = "Deployment"
	KindStatefulSet = "StatefulSet"
	KindDaemonSet   = "DaemonSet"
	KindUnknown     = "Unknown"
)

type Object interface {
	runtime.Object
	metav1.Object
}

func kindOf(obj Object) string {
	switch obj.(type) {
	case *corev1.ConfigMap:
		return KindConfigMap
	case *corev1.Secret:
		return KindSecret
	case *deployment:
		return KindDeployment
	case *statefulset:
		return KindStatefulSet
	case *daemonset:
		return KindDaemonSet
	default:
		return KindUnknown
	}
}

func ObjectKey(obj Object) string {
	return GenKey(kindOf(obj), obj.GetName())
}

func GenKey(kind, name string) string {
	return strings.Join([]string{kind, name}, "/")
}

func ParseKey(key string) (string, string) {
	fields := strings.Split(key, "/")
	if len(fields) != 2 {
		return "", ""
	} else {
		return fields[0], fields[1]
	}
}

type PodController interface {
	runtime.Object
	metav1.Object
	GetObject() runtime.Object
	GetPodTemplate() *corev1.PodTemplateSpec
	SetPodTemplate(*corev1.PodTemplateSpec)
	DeepCopy() PodController
}

type deployment struct {
	*appsv1.Deployment
}

func (d *deployment) GetObject() runtime.Object {
	return d.Deployment
}

func (d *deployment) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.Deployment.Spec.Template
}

func (d *deployment) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.Deployment.Spec.Template = *template
}

func (d *deployment) DeepCopy() PodController {
	return &deployment{d.Deployment.DeepCopy()}
}

type statefulset struct {
	*appsv1.StatefulSet
}

func (d *statefulset) GetObject() runtime.Object {
	return d.StatefulSet
}

func (d *statefulset) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.StatefulSet.Spec.Template
}

func (d *statefulset) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.StatefulSet.Spec.Template = *template
}

func (d *statefulset) DeepCopy() PodController {
	return &statefulset{d.StatefulSet.DeepCopy()}
}

type daemonset struct {
	*appsv1.DaemonSet
}

func (d *daemonset) GetObject() runtime.Object {
	return d.DaemonSet
}

func (d *daemonset) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.DaemonSet.Spec.Template
}

func (d *daemonset) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.DaemonSet.Spec.Template = *template
}

func (d *daemonset) DeepCopy() PodController {
	return &daemonset{d.DaemonSet.DeepCopy()}
}

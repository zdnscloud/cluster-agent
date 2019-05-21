package testutil

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ut "github.com/zdnscloud/cement/unittest"
)

func newPod(index int, ns string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("pod-%v", index), Namespace: ns},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
	}
}

func newPodList(count int) *corev1.PodList {
	var pods []corev1.Pod
	for i := 0; i < count; i++ {
		pods = append(pods, *newPod(i, "default"))
	}

	return &corev1.PodList{
		Items: pods,
	}
}

func TestGetAndList(t *testing.T) {
	m := &MockCache{}

	m.SetGetResult(newPod(1, "network"))
	m.SetListResult(newPodList(3))

	k8spods := corev1.PodList{}
	err := m.List(context.TODO(), nil, &k8spods)
	ut.Assert(t, err == nil, "")

	ut.Equal(t, len(k8spods.Items), 3)
	ut.Equal(t, k8spods.Items[0].Name, "pod-0")
}

package helper

import (
	"context"
	ut "github.com/zdnscloud/cement/unittest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestYamlDocParse(t *testing.T) {
	cases := []struct {
		yaml         string
		expectedDocs []string
	}{
		{
			`
---
---
good
---
apiVersion: v1
data:
  notary-signer-ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIDAzCCAeugAwIBAgIRAO5ZGsMfcfIrCCx93tA0QdwwDQYJKoZIhvcNAQELBQAw
---`,
			[]string{"good",
				`apiVersion: v1
data:
  notary-signer-ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIDAzCCAeugAwIBAgIRAO5ZGsMfcfIrCCx93tA0QdwwDQYJKoZIhvcNAQELBQAw`},
		},
	}

	for _, tc := range cases {
		var docs []string
		mapOnYamlDocument(tc.yaml, func(doc []byte) error {
			docs = append(docs, string(doc))
			return nil
		})

		ut.Equal(t, len(docs), len(tc.expectedDocs))
		for i, doc := range docs {
			ut.Equal(t, doc, tc.expectedDocs[i])
		}
	}
}

func TestYamlObjectParse(t *testing.T) {
	yaml := `
# Generated from 'kube-scheduler.rules' group from https://raw.githubusercontent.com/coreos/kube-prometheus/master/manifests/prometheus-rules.yaml
# Do not change in-place! In order to change this file first read following link:
# https://github.com/helm/charts/tree/master/stable/prometheus-operator/hack
---
apiVersion: v1
kind: Pod
metadata:
  name: counter
spec:
  containers:
  - name: counter
    image: bikecn81/counter
    ports:
    - containerPort: 8888
`

	objects := []runtime.Object{}
	err := MapOnRuntimeObject(yaml, func(ctx_ context.Context, obj runtime.Object) error {
		objects = append(objects, obj)
		return nil
	})
	ut.Assert(t, err == nil, "")
	ut.Equal(t, len(objects), 1)
	pod, ok := objects[0].(*corev1.Pod)
	ut.Assert(t, ok, "should uncode to a pod")
	ut.Equal(t, pod.Name, "counter")
}

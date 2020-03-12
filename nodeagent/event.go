package nodeagent

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/zdnscloud/cement/randomdata"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	eventNamespace = "zcloud"
	eventLevel     = "Warning"
	eventKind      = "Pod"
	eventReason    = "core component abnormal"
)

func CreateEvent(node string, e error) error {
	config, err := config.GetConfig()
	if err != nil {
		return err
	}
	cli, err := client.New(config, client.Options{})
	if err != nil {
		return err
	}
	podName := os.Getenv("POD_NAME")
	message := fmt.Sprintf("abnormality in the communication between core components %s and node-agent on %s, which may cause some functions to not work properly. Err: %v", podName, node, e)
	Event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%s", podName, randomdata.RandString(16)),
			Namespace: eventNamespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      eventKind,
			Namespace: eventNamespace,
			Name:      podName,
		},
		Type:          eventLevel,
		Reason:        eventReason,
		LastTimestamp: metav1.Time{time.Now()},
		Message:       message,
	}
	return cli.Create(context.TODO(), Event)
}

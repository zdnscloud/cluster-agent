package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/zdnscloud/cement/randomdata"
	"github.com/zdnscloud/cluster-agent/monitor/cluster"
	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/cluster-agent/monitor/namespace"
	"github.com/zdnscloud/cluster-agent/monitor/node"
	"github.com/zdnscloud/cluster-agent/storage"
	"github.com/zdnscloud/gok8s/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	checkInterval  = 60
	checkThreshold = 0.0
	eventNamespace = "zcloud"
	eventLevel     = "Warning"
	eventReason    = "resource shortage"
	eventKind      = "Pod"
	eventName      = "cluster-agent"
)

type MonitorManager struct {
	cli       client.Client
	stopCh    chan struct{}
	eventCh   chan interface{}
	Cluster   Monitor
	Node      Monitor
	Namespace Monitor
}

type Monitor interface {
	Start()
}

func NewMonitorManager(cli client.Client, storageMgr *storage.StorageManager) *MonitorManager {
	m := &MonitorManager{
		cli:     cli,
		stopCh:  make(chan struct{}),
		eventCh: make(chan interface{}),
	}
	m.monitorInit(storageMgr)
	return m
}

func (m *MonitorManager) monitorInit(storageMgr *storage.StorageManager) {
	m.Cluster = cluster.New(m.cli, m.stopCh, m.eventCh, checkInterval, checkThreshold)
	m.Node = node.New(m.cli, m.stopCh, m.eventCh, checkInterval, checkThreshold)
	m.Namespace = namespace.New(m.cli, m.stopCh, m.eventCh, storageMgr, checkInterval, checkThreshold)
}

func (m *MonitorManager) Start() {
	go m.Node.Start()
	go m.Cluster.Start()
	go m.Namespace.Start()
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		v := <-m.eventCh
		e := v.(event.Event)
		fmt.Println("=========", e)
		creatK8sEvent(m.cli, e)
	}
}

func (m *MonitorManager) Stop() {
	m.stopCh <- struct{}{}
	<-m.stopCh
	close(m.stopCh)
}

func creatK8sEvent(cli client.Client, e event.Event) {
	if len(e.Namespace) == 0 {
		e.Namespace = eventNamespace
	}
	k8sEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name + "." + randomdata.RandString(16),
			Namespace: e.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      string(e.Kind),
			Namespace: e.Namespace,
			Name:      e.Name,
		},
		Type:          eventLevel,
		Reason:        eventReason,
		LastTimestamp: metav1.Time{time.Now()},
		Message:       e.Message,
	}
	if err := cli.Create(context.TODO(), k8sEvent); err != nil {
		fmt.Println(err)
	}
}

package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/randomdata"
	"github.com/zdnscloud/cluster-agent/monitor/cluster"
	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/cluster-agent/monitor/namespace"
	"github.com/zdnscloud/cluster-agent/monitor/node"
	"github.com/zdnscloud/cluster-agent/storage"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/predicate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	eventNamespace = "zcloud"
	eventLevel     = "Warning"
	eventReason    = "resource shortage"
	eventKind      = "Pod"
	eventName      = "cluster-agent"
)

var ctx = context.TODO()

type MonitorManager struct {
	lock                  sync.RWMutex
	cache                 cache.Cache
	cli                   client.Client
	stopCh                chan struct{}
	EventCh               chan interface{}
	clusterConfig         *event.ClusterMonitorConfig
	namespaceConfig       *event.NamespaceMonitorConfig
	Cluster               Monitor
	Node                  Monitor
	Namespace             Monitor
	startNamespaceMonitor bool
	startClusterMonitor   bool
}

type Monitor interface {
	Start(event.MonitorConfig)
	Stop()
}

func NewMonitorManager(c cache.Cache, cli client.Client, storageMgr *storage.StorageManager) *MonitorManager {
	eventCh := make(chan interface{})
	stopCh := make(chan struct{})
	m := &MonitorManager{
		cache:           c,
		cli:             cli,
		EventCh:         eventCh,
		stopCh:          stopCh,
		clusterConfig:   &event.ClusterMonitorConfig{},
		namespaceConfig: &event.NamespaceMonitorConfig{},
	}
	m.Cluster = cluster.New(cli, eventCh)
	m.Node = node.New(cli, eventCh)
	m.Namespace = namespace.New(cli, storageMgr, eventCh)
	ctrl := controller.New("resource-threshold", c, scheme.Scheme)
	ctrl.Watch(&corev1.ConfigMap{})
	go ctrl.Start(stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	return m
}

func (m *MonitorManager) Stop() {
	m.stopCh <- struct{}{}
	close(m.stopCh)
}

func (m *MonitorManager) Start() {
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		v := <-m.EventCh
		e := v.(event.Event)
		fmt.Println("=========", e)
		creatK8sEvent(m.cli, e)
	}
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
	if err := cli.Create(ctx, k8sEvent); err != nil {
		log.Warnf("Create event failed:%s", err.Error())
	}
}

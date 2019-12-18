package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/cluster-agent/monitor/namespace"
	"github.com/zdnscloud/cluster-agent/monitor/node"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/zke/types"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"time"
)

const (
	clusterConnfigMapNamespace = "kube-system"
	clusterConnfigMapName      = "cluster-config"
)

type Monitor struct {
	cli       client.Client
	stopCh    chan struct{}
	eventCh   chan interface{}
	Interval  int
	Threshold float32
}

type Cluster struct {
	Name        string
	Cpu         int64
	CpuUsed     int64
	Memory      int64
	MemoryUsed  int64
	Pod         int64
	PodUsed     int64
	StorageInfo map[string]event.StorageSize
}

func New(cli client.Client, sch chan struct{}, ech chan interface{}, interval int, threshold float32) *Monitor {
	return &Monitor{
		cli:       cli,
		stopCh:    sch,
		eventCh:   ech,
		Interval:  interval,
		Threshold: threshold,
	}
}

func (m *Monitor) Start() {
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		cluster := getCluster(m.cli)
		m.check(cluster)
		time.Sleep(time.Duration(m.Interval) * time.Second)
	}
}

func (m *Monitor) check(cluster *Cluster) {
	if cluster.Cpu > 0 {
		if ratio := float32(cluster.CpuUsed) / float32(cluster.Cpu); ratio > m.Threshold {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Name:    cluster.Name,
				Message: fmt.Sprintf("High cpu utilization %.2f in cluster", ratio),
			}
		}
	}
	if cluster.Memory > 0 {
		if ratio := float32(cluster.MemoryUsed) / float32(cluster.Memory); ratio > m.Threshold {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Name:    cluster.Name,
				Message: fmt.Sprintf("High memory utilization %.2f in cluster", ratio),
			}
		}
	}
	if cluster.Pod > 0 {
		if ratio := float32(cluster.PodUsed) / float32(cluster.Pod); ratio > m.Threshold {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Name:    cluster.Name,
				Message: fmt.Sprintf("High pod utilization %.2f in cluster", ratio),
			}
		}
	}
	for name, size := range cluster.StorageInfo {
		if ratio := float32(size.Used) / float32(size.Total); ratio > m.Threshold {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Name:    cluster.Name,
				Message: fmt.Sprintf("High storage utilization %.2f for storage type %s in cluster", ratio, name),
			}
		}
	}
}

func getCluster(cli client.Client) *Cluster {
	var cluster Cluster
	cluster.Name = getClusterName(cli)
	cluster.StorageInfo = namespace.GetStorage(cli)
	nodes := node.GetNodes(cli)
	for _, node := range nodes {
		cluster.Cpu += node.Cpu
		cluster.CpuUsed += node.CpuUsed
		cluster.Memory += node.Memory
		cluster.MemoryUsed += node.MemoryUsed
		cluster.Pod += node.Pod
		cluster.PodUsed += node.PodUsed
	}
	return &cluster
}

func getClusterName(cli client.Client) string {
	cm := corev1.ConfigMap{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{clusterConnfigMapNamespace, clusterConnfigMapName}, &cm)
	if err != nil {
		return ""
	}
	var cfg types.ZKEConfig
	json.Unmarshal([]byte(cm.Data[clusterConnfigMapName]), &cfg)
	return cfg.ClusterName
}

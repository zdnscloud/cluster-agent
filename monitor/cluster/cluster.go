package cluster

import (
	"fmt"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/cluster-agent/monitor/namespace"
	"github.com/zdnscloud/cluster-agent/monitor/node"
	"github.com/zdnscloud/gok8s/client"
)

type Monitor struct {
	cli     client.Client
	stopCh  chan struct{}
	eventCh chan interface{}
}

type Cluster struct {
	Cpu         int64
	CpuUsed     int64
	Memory      int64
	MemoryUsed  int64
	Pod         int64
	PodUsed     int64
	StorageInfo map[string]event.StorageSize
}

func New(cli client.Client, ch chan interface{}) *Monitor {
	return &Monitor{
		cli:     cli,
		stopCh:  make(chan struct{}),
		eventCh: ch,
	}
}

func (m *Monitor) Stop() {
	log.Infof("stop cluster monitor")
	m.stopCh <- struct{}{}
	<-m.stopCh
	close(m.stopCh)
}

func (m *Monitor) Start(cfg event.MonitorConfig) {
	log.Infof("start cluster monitor")
	c := cfg.(*event.ClusterMonitorConfig)
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		cluster := getCluster(m.cli)
		m.check(cluster, c)
		time.Sleep(time.Duration(event.CheckInterval) * time.Second)
	}
}

func (m *Monitor) check(cluster *Cluster, cfg *event.ClusterMonitorConfig) {
	if cluster.Cpu > 0 && cfg.Cpu > 0 {
		if ratio := float32(cluster.CpuUsed) / float32(cluster.Cpu); ratio > cfg.Cpu {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Message: fmt.Sprintf("High cpu utilization %.2f in cluster", ratio),
			}
		}
	}
	if cluster.Memory > 0 && cfg.Memory > 0 {
		if ratio := float32(cluster.MemoryUsed) / float32(cluster.Memory); ratio > cfg.Memory {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Message: fmt.Sprintf("High memory utilization %.2f in cluster", ratio),
			}
		}
	}
	if cluster.Pod > 0 && cfg.PodCount > 0 {
		if ratio := float32(cluster.PodUsed) / float32(cluster.Pod); ratio > cfg.PodCount {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Message: fmt.Sprintf("High pod utilization %.2f in cluster", ratio),
			}
		}
	}
	if cfg.Storage > 0 {
		for name, size := range cluster.StorageInfo {
			if ratio := float32(size.Used) / float32(size.Total); ratio > cfg.Storage {
				m.eventCh <- event.Event{
					Kind:    event.ClusterKind,
					Message: fmt.Sprintf("High storage utilization %.2f for storage type %s in cluster", ratio, name),
				}
			}
		}
	}
}

func getCluster(cli client.Client) *Cluster {
	var cluster Cluster
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

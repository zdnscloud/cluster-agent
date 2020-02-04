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
}

func (m *Monitor) Start(cfg *event.MonitorConfig) {
	log.Infof("start cluster monitor")
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		cluster := getCluster(m.cli)
		m.check(cluster, cfg)
		time.Sleep(time.Duration(event.CheckInterval) * time.Second)
	}
}

func (m *Monitor) check(cluster *Cluster, cfg *event.MonitorConfig) {
	if cluster.Cpu > 0 && cfg.Cpu > 0 {
		if ratio := (cluster.CpuUsed * event.Denominator) / cluster.Cpu; ratio > cfg.Cpu {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Message: fmt.Sprintf("High cpu utilization %d%% in cluster", ratio),
			}
			log.Infof("The cpu utilization of the cluster is %d%%, higher than the threshold set by the user %d%%", ratio, cfg.Cpu)
		}
	}
	if cluster.Memory > 0 && cfg.Memory > 0 {
		if ratio := (cluster.MemoryUsed * event.Denominator) / cluster.Memory; ratio > cfg.Memory {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Message: fmt.Sprintf("High memory utilization %d%% in cluster", ratio),
			}
			log.Infof("The memory utilization of the cluster is %d%%, higher than the threshold set by the user %d%%", ratio, cfg.Memory)
		}
	}
	if cluster.Pod > 0 && cfg.PodCount > 0 {
		if ratio := (cluster.PodUsed * event.Denominator) / cluster.Pod; ratio > cfg.PodCount {
			m.eventCh <- event.Event{
				Kind:    event.ClusterKind,
				Message: fmt.Sprintf("High podcount utilization %d%% in cluster", ratio),
			}
			log.Infof("The podcount utilization of the cluster is %d%%, higher than the threshold set by the user %d%%", ratio, cfg.PodCount)
		}
	}
	if cfg.Storage > 0 {
		for name, size := range cluster.StorageInfo {
			if size.Total > 0 {
				if ratio := (size.Used * event.Denominator) / size.Total; ratio > cfg.Storage {
					m.eventCh <- event.Event{
						Kind:    event.ClusterKind,
						Message: fmt.Sprintf("High storage utilization %d%% for storage type %s in cluster", ratio, name),
					}
					log.Infof("The storage utilization of the type %s is %d%%, higher than the threshold set by the user %d%%", name, ratio, cfg.Storage)
				}
			}
		}
	}
}

func getCluster(cli client.Client) *Cluster {
	var cluster Cluster
	cluster.StorageInfo = namespace.GetStorage(cli)
	for _, node := range node.GetNodes(cli) {
		cluster.Cpu += node.Cpu
		cluster.CpuUsed += node.CpuUsed
		cluster.Memory += node.Memory
		cluster.MemoryUsed += node.MemoryUsed
		cluster.Pod += node.Pod
		cluster.PodUsed += node.PodUsed
	}
	return &cluster
}

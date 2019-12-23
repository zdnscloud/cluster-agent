package namespace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/cluster-agent/monitor/node"
	"github.com/zdnscloud/cluster-agent/storage"
	"github.com/zdnscloud/gok8s/client"
	"k8s.io/apimachinery/pkg/labels"
)

var ctx = context.TODO()

type Monitor struct {
	cli            client.Client
	stopCh         chan struct{}
	eventCh        chan interface{}
	PVInfo         map[string]event.StorageSize
	StorageManager *storage.StorageManager
	alreadyRunning bool
}

type Namespace struct {
	Name        string
	Cpu         int64
	CpuUsed     int64
	Memory      int64
	MemoryUsed  int64
	Storage     int64
	StorageUsed int64
}

func New(cli client.Client, storageMgr *storage.StorageManager, ch chan interface{}) *Monitor {
	return &Monitor{
		cli:            cli,
		stopCh:         make(chan struct{}),
		eventCh:        ch,
		PVInfo:         make(map[string]event.StorageSize),
		StorageManager: storageMgr,
	}
}

func (m *Monitor) Stop() {
	log.Infof("stop namespace monitor")
	m.stopCh <- struct{}{}
	<-m.stopCh
	close(m.stopCh)
}

func (m *Monitor) Start(cfg event.MonitorConfig) {
	c := cfg.(*event.NamespaceMonitorConfig)
	if m.alreadyRunning {
		return
	}
	m.alreadyRunning = true
	log.Infof("start namespace monitor")
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		m.genPVInfo()
		for ns, cfg := range c.Configs {
			namespace := getNamespace(m.cli, ns, m.PVInfo)
			m.check(namespace, cfg)
			m.checkPodStorgeUsed(namespace, cfg)
		}
		time.Sleep(time.Duration(event.CheckInterval) * time.Second)
	}
}

func (m *Monitor) genPVInfo() {
	mountpoints := m.StorageManager.GetBuf()
	if len(mountpoints) == 0 {
		mountpoints = m.StorageManager.SetBuf()
	}
	for mountpoint, size := range mountpoints {
		pv := strings.Split(mountpoint, "/")[8]
		m.PVInfo[pv] = event.StorageSize{
			Total: size[0],
			Used:  size[1],
		}
	}
}

func (m *Monitor) check(namespace *Namespace, cfg *event.Config) {
	if namespace.Cpu > 0 && cfg.Cpu > 0 {
		if ratio := float32(namespace.CpuUsed) / float32(namespace.Cpu); ratio > cfg.Cpu {
			m.eventCh <- event.Event{
				Namespace: namespace.Name,
				Kind:      event.NamespaceKind,
				Name:      namespace.Name,
				Message:   fmt.Sprintf("High cpu utilization %.2f%%", ratio*100),
			}
			log.Infof("The CPU utilization of namespace %s is %.2f%%, higher than the indicator set by the user %.2f%%", namespace.Name, ratio*100, cfg.Cpu*100)
		}
	}
	if namespace.Memory > 0 && cfg.Memory > 0 {
		if ratio := float32(namespace.MemoryUsed) / float32(namespace.Memory); ratio > cfg.Memory {
			m.eventCh <- event.Event{
				Namespace: namespace.Name,
				Kind:      event.NamespaceKind,
				Name:      namespace.Name,
				Message:   fmt.Sprintf("High memory utilization %.2f%%", ratio*100),
			}
			log.Infof("The memory utilization of namespace %s is %.2f%%, higher than the indicator set by the user %.2f%%", namespace.Name, ratio*100, cfg.Memory*100)
		}
	}
	if namespace.Storage > 0 && cfg.Storage > 0 {
		if ratio := float32(namespace.StorageUsed) / float32(namespace.Storage); ratio > cfg.Storage {
			m.eventCh <- event.Event{
				Namespace: namespace.Name,
				Kind:      event.NamespaceKind,
				Name:      namespace.Name,
				Message:   fmt.Sprintf("High storage utilization %.2f%%", ratio*100),
			}
			log.Infof("The storage utilization of namespace %s is %.2f%%, higher than the indicator set by the user %.2f%%", namespace.Name, ratio*100, cfg.Storage*100)
		}
	}
}

func (m *Monitor) checkPodStorgeUsed(namespace *Namespace, cfg *event.Config) {
	pods := getPodsWithPvcs(m.cli, namespace.Name)
	pvcs := getPvcsWithPv(m.cli, namespace.Name)
	for pod, ps := range pods {
		for _, pvc := range ps {
			pv, ok := pvcs[pvc]
			if ok {
				size, ok := m.PVInfo[pv]
				if ok && cfg.PodStorage > 0 {
					if ratio := float32(size.Used) / float32(size.Total); ratio > cfg.PodStorage {
						m.eventCh <- event.Event{
							Namespace: namespace.Name,
							Kind:      event.PodKind,
							Name:      pod,
							Message:   fmt.Sprintf("High storage utilization %.2f%%", ratio*100),
						}
						log.Infof("The sorage utilization of pod %s is %.2f%%, higher than the threshold set by the user %.2f%%", pod, ratio*100, cfg.PodStorage*100)
					}
				}
			}
		}
	}
}

func getNamespace(cli client.Client, ns string, pvInfo map[string]event.StorageSize) *Namespace {
	var namespace Namespace
	namespace.Name = ns
	podMetricsList, err := cli.GetPodMetrics(ns, "", labels.Everything())
	if err != nil {
		log.Warnf("Get pod metrics failed:%s", err.Error())
	}
	for _, pod := range podMetricsList.Items {
		var cpuUsed, memoryUsed int64
		for _, container := range pod.Containers {
			cpuUsed += container.Usage.Cpu().MilliValue()
			memoryUsed += container.Usage.Memory().Value()
		}
		namespace.CpuUsed += cpuUsed
		namespace.MemoryUsed += memoryUsed
	}
	namespace.StorageUsed = getAllPVCUsedSize(cli, ns, pvInfo)

	var cpu, mem, storagesize int64
	nodes := node.GetNodes(cli)
	for _, n := range nodes {
		cpu += n.Cpu
		mem += n.Memory
	}
	storage := GetStorage(cli)
	for _, size := range storage {
		storagesize += size.Total
	}

	cpuquota, memquota, storagequota := getQuotas(cli, ns)
	if cpuquota != 0 {
		namespace.Cpu = cpuquota
	} else {
		namespace.Cpu = cpu
	}
	if memquota != 0 {
		namespace.Memory = memquota
	} else {
		namespace.Memory = mem
	}
	if storagequota != 0 {
		namespace.Storage = storagequota
	} else {
		namespace.Storage = storagesize
	}
	return &namespace
}

func getAllPVCUsedSize(cli client.Client, ns string, pvInfo map[string]event.StorageSize) int64 {
	var used int64
	pvcs := getPvcsWithPv(cli, ns)
	for _, pv := range pvcs {
		size, ok := pvInfo[pv]
		if ok {
			used += size.Used
		}
	}
	return used * 1024
}

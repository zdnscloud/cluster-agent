package namespace

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

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
	StorageManager *storage.StorageManager
}

type Namespace struct {
	Name        string
	Cpu         int64
	CpuUsed     int64
	Memory      int64
	MemoryUsed  int64
	Storage     int64
	StorageUsed int64
	PvInfo      map[string]event.StorageSize
}

func New(cli client.Client, storageMgr *storage.StorageManager, ch chan interface{}) *Monitor {
	return &Monitor{
		cli:            cli,
		stopCh:         make(chan struct{}),
		eventCh:        ch,
		StorageManager: storageMgr,
	}
}

func (m *Monitor) Stop() {
	log.Infof("stop namespace monitor")
	m.stopCh <- struct{}{}
	<-m.stopCh
}

func (m *Monitor) Start(cfg *event.MonitorConfig) {
	log.Infof("start namespace monitor")
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		pvInfo := m.genPVInfo()
		namespaces := corev1.NamespaceList{}
		if err := m.cli.List(context.TODO(), nil, &namespaces); err != nil {
			continue
		}
		for _, ns := range namespaces.Items {
			namespace, err := getNamespace(m.cli, ns.Name, pvInfo)
			if err != nil {
				continue
			}
			m.check(namespace, cfg)
			m.checkPodStorgeUsed(namespace, cfg)
		}
		time.Sleep(time.Duration(event.CheckInterval) * time.Second)
	}
}

func (m *Monitor) genPVInfo() map[string]event.StorageSize {
	mountpoints := m.StorageManager.GetBuf()
	if len(mountpoints) == 0 {
		mountpoints = m.StorageManager.SetBuf()
	}
	pvInfo := make(map[string]event.StorageSize)
	for mountpoint, size := range mountpoints {
		pv := strings.Split(mountpoint, "/")[8]
		pvInfo[pv] = event.StorageSize{
			Total: size[0],
			Used:  size[1],
		}
	}
	return pvInfo
}

func (m *Monitor) check(namespace *Namespace, cfg *event.MonitorConfig) {
	if namespace.Cpu > 0 && cfg.Cpu > 0 {
		if ratio := (namespace.CpuUsed * event.Denominator) / namespace.Cpu; ratio > cfg.Cpu {
			m.eventCh <- event.Event{
				Namespace: namespace.Name,
				Kind:      event.NamespaceKind,
				Name:      namespace.Name,
				Message:   fmt.Sprintf("High cpu utilization %d%%", ratio),
			}
			log.Infof("The CPU utilization of namespace %s is %d%%, higher than the threshold set by the user %d%%", namespace.Name, ratio, cfg.Cpu)
		}
	}
	if namespace.Memory > 0 && cfg.Memory > 0 {
		if ratio := (namespace.MemoryUsed * event.Denominator) / namespace.Memory; ratio > cfg.Memory {
			m.eventCh <- event.Event{
				Namespace: namespace.Name,
				Kind:      event.NamespaceKind,
				Name:      namespace.Name,
				Message:   fmt.Sprintf("High memory utilization %d%%", ratio),
			}
			log.Infof("The memory utilization of namespace %s is %d%%, higher than the threshold set by the user %d%%", namespace.Name, ratio, cfg.Memory)
		}
	}
	if namespace.Storage > 0 && cfg.Storage > 0 {
		if ratio := (namespace.StorageUsed * event.Denominator) / namespace.Storage; ratio > cfg.Storage {
			m.eventCh <- event.Event{
				Namespace: namespace.Name,
				Kind:      event.NamespaceKind,
				Name:      namespace.Name,
				Message:   fmt.Sprintf("High storage utilization %d%%", ratio),
			}
			log.Infof("The storage utilization of namespace %s is %d%%, higher than the threshold set by the user %d%%", namespace.Name, ratio, cfg.Storage)
		}
	}
}

func (m *Monitor) checkPodStorgeUsed(namespace *Namespace, cfg *event.MonitorConfig) {
	pods := getPodsWithPvcs(m.cli, namespace.Name)
	pvcs := getPvcsWithPv(m.cli, namespace.Name)
	for pod, ps := range pods {
		for _, pvc := range ps {
			if pv, ok := pvcs[pvc]; ok {
				size, ok := namespace.PvInfo[pv]
				if ok && size.Total > 0 && cfg.Storage > 0 {
					if ratio := (size.Used * event.Denominator) / size.Total; ratio > cfg.Storage {
						m.eventCh <- event.Event{
							Namespace: namespace.Name,
							Kind:      event.PodKind,
							Name:      pod,
							Message:   fmt.Sprintf("High storage utilization %d%%", ratio),
						}
						log.Infof("The sorage utilization of pod %s is %d%%, higher than the threshold set by the user %d%%", pod, ratio, cfg.Storage)
					}
				}
			}
		}
	}
}

func getNamespace(cli client.Client, ns string, pvInfo map[string]event.StorageSize) (*Namespace, error) {
	var namespace Namespace
	namespace.Name = ns
	namespace.PvInfo = pvInfo
	podMetricsList, err := cli.GetPodMetrics(ns, "", labels.Everything())
	if err != nil {
		log.Warnf("Get pod metrics failed:%s", err.Error())
		return nil, err
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
	return &namespace, nil
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

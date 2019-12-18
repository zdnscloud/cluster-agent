package namespace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/cluster-agent/monitor/node"
	"github.com/zdnscloud/cluster-agent/storage"
	"github.com/zdnscloud/gok8s/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var ctx = context.TODO()

type Monitor struct {
	cli            client.Client
	stopCh         chan struct{}
	eventCh        chan interface{}
	PVInfo         map[string]event.StorageSize
	StorageManager *storage.StorageManager
	Interval       int
	Threshold      float32
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

func New(cli client.Client, sch chan struct{}, ech chan interface{}, storageMgr *storage.StorageManager, interval int, threshold float32) *Monitor {
	return &Monitor{
		cli:            cli,
		stopCh:         sch,
		eventCh:        ech,
		PVInfo:         make(map[string]event.StorageSize),
		StorageManager: storageMgr,
		Interval:       interval,
		Threshold:      threshold,
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
		m.genPVInfo()
		namespaces := getNamespaces(m.cli, m.PVInfo)
		m.check(namespaces)
		m.checkPodStorgeUsed(namespaces)
		time.Sleep(time.Duration(m.Interval) * time.Second)
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

func getNamespaces(cli client.Client, pvInfo map[string]event.StorageSize) []*Namespace {
	var ns []*Namespace
	namespaces := corev1.NamespaceList{}
	_ = cli.List(ctx, nil, &namespaces)
	for _, namespace := range namespaces.Items {
		ns = append(ns, getNamespace(cli, namespace.Name, pvInfo))
	}
	return ns
}

func (m *Monitor) check(namespaces []*Namespace) {
	for _, namespace := range namespaces {
		if namespace.Cpu > 0 {
			if ratio := float32(namespace.CpuUsed) / float32(namespace.Cpu); ratio > m.Threshold {
				m.eventCh <- event.Event{
					Namespace: namespace.Name,
					Kind:      event.NamespaceKind,
					Name:      namespace.Name,
					Message:   fmt.Sprintf("High cpu utilization %.2f in namespace %s", ratio, namespace.Name),
				}
			}
		}
		if namespace.Memory > 0 {
			if ratio := float32(namespace.MemoryUsed) / float32(namespace.Memory); ratio > m.Threshold {
				m.eventCh <- event.Event{
					Namespace: namespace.Name,
					Kind:      event.NamespaceKind,
					Name:      namespace.Name,
					Message:   fmt.Sprintf("High memory utilization %.2f in namespace %s", ratio, namespace.Name),
				}
			}
		}
		if namespace.Storage > 0 {
			if ratio := float32(namespace.StorageUsed) / float32(namespace.Storage); ratio > m.Threshold {
				m.eventCh <- event.Event{
					Namespace: namespace.Name,
					Kind:      event.NamespaceKind,
					Name:      namespace.Name,
					Message:   fmt.Sprintf("High storage utilization %.2f in namespace %s", ratio, namespace.Name),
				}
			}
		}
	}
}

func (m *Monitor) checkPodStorgeUsed(namespaces []*Namespace) {
	for _, namespace := range namespaces {
		pods := getPodsWithPvcs(m.cli, namespace.Name)
		pvcs := getPvcsWithPv(m.cli, namespace.Name)
		for pod, ps := range pods {
			for _, pvc := range ps {
				pv, ok := pvcs[pvc]
				if ok {
					size, ok := m.PVInfo[pv]
					if ok {
						if ratio := float32(size.Used) / float32(size.Total); ratio > m.Threshold {
							m.eventCh <- event.Event{
								Namespace: namespace.Name,
								Kind:      event.PodKind,
								Name:      pod,
								Message:   fmt.Sprintf("High storage utilization %.2f for pod %s in namespace %s", ratio, pod, namespace.Name),
							}
						}
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
	return used
}

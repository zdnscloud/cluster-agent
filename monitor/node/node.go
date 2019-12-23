package node

import (
	"context"
	"fmt"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/gok8s/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
)

type Monitor struct {
	cli     client.Client
	stopCh  chan struct{}
	eventCh chan interface{}
}

type Node struct {
	Name       string
	Cpu        int64
	CpuUsed    int64
	Memory     int64
	MemoryUsed int64
	Pod        int64
	PodUsed    int64
}

func New(cli client.Client, ch chan interface{}) *Monitor {
	return &Monitor{
		cli:     cli,
		stopCh:  make(chan struct{}),
		eventCh: ch,
	}
}

func (m *Monitor) Stop() {
	log.Infof("stop node monitor")
	m.stopCh <- struct{}{}
	<-m.stopCh
	close(m.stopCh)
}

func (m *Monitor) Start(cfg event.MonitorConfig) {
	log.Infof("start node monitor")
	c := cfg.(*event.ClusterMonitorConfig)
	for {
		select {
		case <-m.stopCh:
			m.stopCh <- struct{}{}
			return
		default:
		}
		nodes := GetNodes(m.cli)
		m.check(nodes, c)
		time.Sleep(time.Duration(event.CheckInterval) * time.Second)
	}
}
func (m *Monitor) check(nodes []*Node, cfg *event.ClusterMonitorConfig) {
	for _, node := range nodes {
		if cfg.NodeCpu > 0 {
			if ratio := (node.CpuUsed*event.Denominator) / (node.Cpu); ratio > (cfg.NodeCpu) {
				m.eventCh <- event.Event{
					Kind:    event.NodeKind,
					Name:    node.Name,
					Message: fmt.Sprintf("High cpu utilization %d%%", ratio),
				}
				log.Infof("The CPU utilization of node %s is %d%%, higher than the threshold set by the user %d%%", node.Name, ratio, cfg.NodeCpu)
			}
		}
		if cfg.NodeMemory > 0 {
			if ratio := (node.MemoryUsed*event.Denominator) / (node.Memory); ratio > (cfg.NodeMemory) {
				m.eventCh <- event.Event{
					Kind:    event.NodeKind,
					Name:    node.Name,
					Message: fmt.Sprintf("High memory utilization %d%%", ratio),
				}
				log.Infof("The memory utilization of node %s is %d%%, higher than the threshold set by the user %d%%", node.Name, ratio, cfg.NodeCpu)
			}
		}
	}
}

func GetNodes(cli client.Client) []*Node {
	var nodes []*Node
	k8sNodes := corev1.NodeList{}
	if err := cli.List(context.TODO(), nil, &k8sNodes); err != nil {
		log.Warnf("Get nodes failed:%s", err.Error())
		return nodes
	}

	podCountOnNode := getPodCountOnNode(cli, "")
	nodeMetrics := getNodeMetrics(cli, "")
	for _, k8sNode := range k8sNodes.Items {
		nodes = append(nodes, k8sNodeToNode(&k8sNode, nodeMetrics, podCountOnNode))
	}
	return nodes
}

func getPodCountOnNode(cli client.Client, name string) map[string]int {
	podCountOnNode := make(map[string]int)

	pods := corev1.PodList{}
	err := cli.List(context.TODO(), nil, &pods)
	if err == nil {
		for _, p := range pods.Items {
			if p.Status.Phase != corev1.PodRunning {
				continue
			}

			n := p.Spec.NodeName
			if name != "" && n != name {
				continue
			}
			podCountOnNode[n] += 1
		}
	} else {
		log.Warnf("Get pods failed:%s", err.Error())
	}
	return podCountOnNode
}

func getNodeMetrics(cli client.Client, name string) map[string]metricsapi.NodeMetrics {
	nodeMetricsByName := make(map[string]metricsapi.NodeMetrics)
	nodeMetricsList, err := cli.GetNodeMetrics(name, labels.Everything())
	if err == nil {
		for _, metrics := range nodeMetricsList.Items {
			nodeMetricsByName[metrics.Name] = metrics
		}
	} else {
		log.Warnf("Get node meterics failed:%s", err.Error())
	}
	return nodeMetricsByName
}

func k8sNodeToNode(k8sNode *corev1.Node, nodeMetrics map[string]metricsapi.NodeMetrics, podCountOnNode map[string]int) *Node {
	status := &k8sNode.Status
	cpuAva := status.Allocatable.Cpu().MilliValue()
	memoryAva := status.Allocatable.Memory().Value()
	podAva := status.Allocatable.Pods().Value()

	usageMetrics := nodeMetrics[k8sNode.Name]
	cpuUsed := usageMetrics.Usage.Cpu().MilliValue()
	memoryUsed := usageMetrics.Usage.Memory().Value()
	podUsed := int64(podCountOnNode[k8sNode.Name])

	return &Node{
		Name:       k8sNode.Name,
		Cpu:        cpuAva,
		CpuUsed:    cpuUsed,
		Memory:     memoryAva,
		MemoryUsed: memoryUsed,
		Pod:        podAva,
		PodUsed:    podUsed,
	}
}

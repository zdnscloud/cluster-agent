package node

import (
	"context"
	"fmt"
	"time"

	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/gok8s/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
)

type Monitor struct {
	cli       client.Client
	stopCh    chan struct{}
	eventCh   chan interface{}
	Interval  int
	Threshold float32
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
		nodes := GetNodes(m.cli)
		m.check(nodes)
		time.Sleep(time.Duration(m.Interval) * time.Second)
	}
}
func (m *Monitor) check(nodes []*Node) {
	for _, node := range nodes {
		if ratio := float32(node.CpuUsed) / float32(node.Cpu); ratio > m.Threshold {
			m.eventCh <- event.Event{
				Kind:    event.NodeKind,
				Name:    node.Name,
				Message: fmt.Sprintf("High cpu utilization %.2f on node %s", ratio, node.Name),
			}
		}
		if ratio := float32(node.MemoryUsed) / float32(node.Memory); ratio > m.Threshold {
			m.eventCh <- event.Event{
				Kind:    event.NodeKind,
				Name:    node.Name,
				Message: fmt.Sprintf("High memory utilization %.2f on node %s", ratio, node.Name),
			}
		}
		if ratio := float32(node.PodUsed) / float32(node.Pod); ratio > m.Threshold {
			m.eventCh <- event.Event{
				Kind:    event.NodeKind,
				Name:    node.Name,
				Message: fmt.Sprintf("High pod utilization %.2f on node %s", ratio, node.Name),
			}
		}
	}
}

func GetNodes(cli client.Client) []*Node {
	var nodes []*Node
	k8sNodes, err := getK8SNodes(cli)
	if err != nil {
		return nodes
	}

	podCountOnNode := getPodCountOnNode(cli, "")
	nodeMetrics := getNodeMetrics(cli, "")
	for _, k8sNode := range k8sNodes.Items {
		nodes = append(nodes, k8sNodeToNode(&k8sNode, nodeMetrics, podCountOnNode))
	}
	return nodes
}

func getK8SNodes(cli client.Client) (*corev1.NodeList, error) {
	nodes := corev1.NodeList{}
	err := cli.List(context.TODO(), nil, &nodes)
	return &nodes, err
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

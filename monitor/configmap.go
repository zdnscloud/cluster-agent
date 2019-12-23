package monitor

import (
	"strconv"
	"strings"

	"github.com/zdnscloud/cement/log"
	monitevent "github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	corev1 "k8s.io/api/core/v1"
)

const (
	ClusterThresholdConfigmapName         = "resource-threshold"
	NamespaceThresholdConfigmapNamePrefix = "resource-threshold-"
	ThresholdConfigmapNamespace           = "zcloud"
	CpuConfigName                         = "cpu"
	MemoryConfigName                      = "memory"
	StorageConfigName                     = "storage"
	PodCountConfigName                    = "podCount"
	PodStorageConfigName                  = "podStorage"
	NodeCpuConfigName                     = "nodeCpu"
	NodeMemoryConfigName                  = "nodeMemory"
)

func (m *MonitorManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.ConfigMap:
		if obj.Name == ClusterThresholdConfigmapName && obj.Namespace == ThresholdConfigmapNamespace {
			m.initClusterMonitorConfig(obj)
			go m.Cluster.Start(m.clusterConfig)
			go m.Node.Start(m.clusterConfig)
		}
		if strings.HasPrefix(obj.Name, NamespaceThresholdConfigmapNamePrefix) && obj.Namespace == ThresholdConfigmapNamespace {
			namespace := strings.TrimPrefix(obj.Name, NamespaceThresholdConfigmapNamePrefix)
			m.initNamespaceMonitorConfig(obj, namespace)
			go m.Namespace.Start(m.namespaceConfig)
		}
	}
	return handler.Result{}, nil
}
func (m *MonitorManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.ObjectNew.(type) {
	case *corev1.ConfigMap:
		if obj.Name == ClusterThresholdConfigmapName && obj.Namespace == ThresholdConfigmapNamespace {
			m.initClusterMonitorConfig(obj)
		}
		if strings.HasPrefix(obj.Name, NamespaceThresholdConfigmapNamePrefix) && obj.Namespace == ThresholdConfigmapNamespace {
			namespace := strings.TrimPrefix(obj.Name, NamespaceThresholdConfigmapNamePrefix)
			m.initNamespaceMonitorConfig(obj, namespace)
		}
	}
	return handler.Result{}, nil
}
func (m *MonitorManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.ConfigMap:
		if obj.Name == ClusterThresholdConfigmapName && obj.Namespace == ThresholdConfigmapNamespace {
			m.Cluster.Stop()
			m.Node.Stop()
		}
		if strings.HasPrefix(obj.Name, NamespaceThresholdConfigmapNamePrefix) && obj.Namespace == ThresholdConfigmapNamespace {
			namespace := strings.TrimPrefix(obj.Name, NamespaceThresholdConfigmapNamePrefix)
			delete(m.namespaceConfig.Configs, namespace)
			if isOnlyOne(m.cli) {
				m.Namespace.Stop()
			}
		}
	}
	return handler.Result{}, nil
}
func (m *MonitorManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func isOnlyOne(cli client.Client) bool {
	cms := corev1.ConfigMapList{}
	if err := cli.List(ctx, &client.ListOptions{Namespace: ThresholdConfigmapNamespace}, &cms); err != nil {
		log.Warnf("Get configmaps failed:%s", err.Error())
		return false
	}
	for _, cm := range cms.Items {
		if strings.HasPrefix(cm.Name, NamespaceThresholdConfigmapNamePrefix) {
			return false
		}
	}
	return true
}

func (m *MonitorManager) initClusterMonitorConfig(cm *corev1.ConfigMap) {
	if v, ok := cm.Data[CpuConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.clusterConfig.Cpu = int64(n)
	}
	if v, ok := cm.Data[MemoryConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.clusterConfig.Memory = int64(n)
	}
	if v, ok := cm.Data[StorageConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.clusterConfig.Storage = int64(n)
	}
	if v, ok := cm.Data[PodCountConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.clusterConfig.PodCount = int64(n)
	}
	if v, ok := cm.Data[NodeCpuConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.clusterConfig.NodeCpu = int64(n)
	}
	if v, ok := cm.Data[NodeMemoryConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.clusterConfig.NodeMemory = int64(n)
	}
	log.Infof("update cluster and node monitor config %v", *m.clusterConfig)
}

func (m *MonitorManager) initNamespaceMonitorConfig(cm *corev1.ConfigMap, namespace string) {
	if m.namespaceConfig.Configs == nil {
		m.namespaceConfig.Configs = make(map[string]*monitevent.Config)
	}
	_, ok := m.namespaceConfig.Configs[namespace]
	if !ok {
		m.namespaceConfig.Configs[namespace] = &monitevent.Config{}
	}
	if v, ok := cm.Data[CpuConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.namespaceConfig.Configs[namespace].Cpu = int64(n)
	}
	if v, ok := cm.Data[MemoryConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.namespaceConfig.Configs[namespace].Memory = int64(n)
	}
	if v, ok := cm.Data[StorageConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.namespaceConfig.Configs[namespace].Storage = int64(n)
	}
	if v, ok := cm.Data[PodStorageConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.namespaceConfig.Configs[namespace].PodStorage = int64(n)
	}
	for ns, cfg := range m.namespaceConfig.Configs {
		log.Infof("update namespace %s monitor config: %v", ns, *cfg)
	}
}

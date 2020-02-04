package monitor

import (
	"strconv"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	corev1 "k8s.io/api/core/v1"
)

const (
	ThresholdConfigmapName      = "threshold"
	ThresholdConfigmapNamespace = "zcloud"
	CpuConfigName               = "cpu"
	MemoryConfigName            = "memory"
	StorageConfigName           = "storage"
	PodCountConfigName          = "podCount"
)

func (m *MonitorManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.ConfigMap:
		if obj.Name == ThresholdConfigmapName && obj.Namespace == ThresholdConfigmapNamespace {
			m.initMonitorConfig(obj)
			go m.Cluster.Start(m.monitorConfig)
			go m.Node.Start(m.monitorConfig)
			go m.Namespace.Start(m.monitorConfig)
		}
	}
	return handler.Result{}, nil
}
func (m *MonitorManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.ObjectNew.(type) {
	case *corev1.ConfigMap:
		if obj.Name == ThresholdConfigmapName && obj.Namespace == ThresholdConfigmapNamespace {
			m.initMonitorConfig(obj)
		}
	}
	return handler.Result{}, nil
}
func (m *MonitorManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch obj := e.Object.(type) {
	case *corev1.ConfigMap:
		if obj.Name == ThresholdConfigmapName && obj.Namespace == ThresholdConfigmapNamespace {
			m.Cluster.Stop()
			m.Node.Stop()
			m.Namespace.Stop()
		}
	}
	return handler.Result{}, nil
}
func (m *MonitorManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (m *MonitorManager) initMonitorConfig(cm *corev1.ConfigMap) {
	if v, ok := cm.Data[CpuConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.monitorConfig.Cpu = int64(n)
	}
	if v, ok := cm.Data[MemoryConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.monitorConfig.Memory = int64(n)
	}
	if v, ok := cm.Data[StorageConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.monitorConfig.Storage = int64(n)
	}
	if v, ok := cm.Data[PodCountConfigName]; ok {
		n, _ := strconv.Atoi(v)
		m.monitorConfig.PodCount = int64(n)
	}
	log.Infof("update monitor config %v", *m.monitorConfig)
}

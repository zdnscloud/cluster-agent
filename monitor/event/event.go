package event

const (
	CheckInterval           = 60
	ClusterKind   EventKind = "cluster"
	NodeKind      EventKind = "node"
	NamespaceKind EventKind = "namespace"
	PodKind       EventKind = "pod"
	Denominator             = 100
)

type Event struct {
	Namespace string
	Kind      EventKind
	Name      string
	Message   string
}

type EventKind string

type MonitorConfig interface{}

type ClusterMonitorConfig struct {
	Cpu        int64
	Memory     int64
	Storage    int64
	PodCount   int64
	NodeCpu    int64
	NodeMemory int64
}

type NamespaceMonitorConfig struct {
	Configs map[string]*Config
}

type Config struct {
	Cpu        int64
	Memory     int64
	Storage    int64
	PodStorage int64
}

type StorageSize struct {
	Total int64
	Used  int64
}

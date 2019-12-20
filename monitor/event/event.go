package event

const (
	CheckInterval           = 60
	ClusterKind   EventKind = "Cluster"
	NodeKind      EventKind = "Node"
	NamespaceKind EventKind = "Namespace"
	PodKind       EventKind = "Pod"
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
	Cpu        float32
	Memory     float32
	Storage    float32
	PodCount   float32
	NodeCpu    float32
	NodeMemory float32
}

type NamespaceMonitorConfig struct {
	Configs map[string]*Config
}

type Config struct {
	Cpu        float32
	Memory     float32
	Storage    float32
	PodStorage float32
}

type StorageSize struct {
	Total int64
	Used  int64
}

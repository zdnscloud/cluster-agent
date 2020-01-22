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

type MonitorConfig struct {
	Cpu      int64
	Memory   int64
	Storage  int64
	PodCount int64
}

type StorageSize struct {
	Total int64
	Used  int64
}

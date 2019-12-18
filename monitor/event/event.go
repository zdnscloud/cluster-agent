package event

const (
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

type StorageSize struct {
	Total int64
	Used  int64
}

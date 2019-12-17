package types

type Stat struct {
	ID              string           `json:"id,omitempty"`
	Resource        Resource         `json:"resource,omitempty"`
	TimeWindow      string           `json:"timeWindow,omitempty"`
	Status          string           `json:"status,omitempty"`
	MeshedPodCount  int              `json:"meshedPodCount,omitempty"`
	RunningPodCount int              `json:"runningPodCount,omitempty"`
	FailedPodCount  int              `json:"failedPodCount,omitempty"`
	BasicStat       BasicStat        `json:"basicStat,omitempty"`
	TcpStat         TcpStat          `json:"tcpStat,omitempty"`
	TsStat          TrafficSplitStat `json:"trafficSplitStat,omitempty"`
	PodErrors       PodErrors        `json:"podErrors,omitempty"`
}

type BasicStat struct {
	SuccessCount       int `json:"successCount,omitempty"`
	FailureCount       int `json:"failureCount,omitempty"`
	LatencyMsP50       int `json:"latencyMsP50,omitempty"`
	LatencyMsP95       int `json:"latencyMsP95,omitempty"`
	LatencyMsP99       int `json:"latencyMsP99,omitempty"`
	ActualSuccessCount int `json:"actualSuccessCount,omitempty"`
	ActualFailureCount int `json:"actualFailureCount,omitempty"`
}

type TcpStat struct {
	OpenConnections int `json:"openConnections,omitempty"`
	ReadBytesTotal  int `json:"readBytesTotal,omitempty"`
	WriteBytesTotal int `json:"writeBytesTotal,omitempty"`
}

type TrafficSplitStat struct {
	Apex   string `json:"apex,omitempty"`
	Leaf   string `json:"leaf,omitempty"`
	Weight string `json:"weight,omitempty"`
}

type PodErrors []PodError
type PodError struct {
	PodName string           `json:"podName,omitempty"`
	Errors  []ContainerError `json:"errors,omitempty"`
}

func (p PodErrors) Len() int {
	return len(p)
}

func (p PodErrors) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p PodErrors) Less(i, j int) bool {
	return p[i].PodName < p[j].PodName
}

type ContainerError struct {
	Message   string `json:"message,omitempty"`
	Container string `json:"container,omitempty"`
	Image     string `json:"image,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Stats []Stat

func (s Stats) Len() int {
	return len(s)
}

func (s Stats) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Stats) Less(i, j int) bool {
	if s[i].Resource.Type == s[j].Resource.Type {
		return s[i].Resource.Namespace+s[i].Resource.Name < s[j].Resource.Namespace+s[j].Resource.Name
	}
	return s[i].Resource.Type < s[j].Resource.Type
}

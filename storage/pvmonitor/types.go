package pvmonitor

import (
	"github.com/zdnscloud/cluster-agent/storage/types"
)

type PVMonitor struct {
	Name      string
	PVs       []types.PV
	PvAndPvc  map[string]PVC
	PvcAndPod map[string][]types.Pod
	PVCAndSc  map[string]string
}

type PVC struct {
	Name       string
	VolumeName string
	Pods       []types.Pod
}

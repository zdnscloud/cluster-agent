package storage

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"unsafe"
)

func getVg(node string) VG {
	vgres, err := GetVG(node)
	if err != nil {
		return VG{}
	}
	for _, v := range vgres {
		if v.Name == DefaultVgName {
			return VG{
				Name:     v.Name,
				Size:     byteToGb(v.Size),
				FreeSize: byteToGb(v.FreeSize),
			}
		}
	}
	return VG{}
}

func byteToGb(num uint64) int {
	f := float64(num) / 1024 / 1024 / 1024
	i, _ := strconv.Atoi(fmt.Sprintf("%.0f", f))
	return i
}

func getLvmNode(k8snode *corev1.Node) Node {
	addr := k8snode.Annotations[ZkeInternalIPAnnKey]
	vg := getVg(addr)
	return Node{
		Name:      k8snode.Name,
		TotalSize: vg.Size,
		FreeSize:  vg.FreeSize,
	}
}

func sizetog(q resource.Quantity) int {
	int64value := q.Value()
	uint64value := (*uint64)(unsafe.Pointer(&int64value))
	pvsize := byteToGb(*uint64value)
	return pvsize
}

package storageclass

import (
	"fmt"
	resource "k8s.io/apimachinery/pkg/api/resource"
	"strconv"
	"unsafe"
)

func sizetog(q resource.Quantity) int {
	int64value := q.Value()
	uint64value := (*uint64)(unsafe.Pointer(&int64value))
	pvsize := byteToGb(*uint64value)
	return pvsize
}

func byteToGb(num uint64) int {
	f := float64(num) / 1024 / 1024 / 1024
	i, _ := strconv.Atoi(fmt.Sprintf("%.0f", f))
	return i
}

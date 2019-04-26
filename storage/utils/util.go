package utils

import (
	"fmt"
	resource "k8s.io/apimachinery/pkg/api/resource"
	"strconv"
)

func SizetoGb(q resource.Quantity) int {
	return ByteToGb(uint64(q.Value()))
}

func ByteToGb(num uint64) int {
	f := float64(num) / (1024 * 1024 * 1024)
	i, _ := strconv.Atoi(fmt.Sprintf("%.0f", f))
	return i
}

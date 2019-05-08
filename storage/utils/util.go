package utils

import (
	"fmt"
	resource "k8s.io/apimachinery/pkg/api/resource"
)

func SizetoGb(q resource.Quantity) string {
	return ByteToGb(uint64(q.Value()))
}

func ByteToGb(num uint64) string {
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", f)
}

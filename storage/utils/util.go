package utils

import (
	"fmt"
	resource "k8s.io/apimachinery/pkg/api/resource"
	"strconv"
)

func SizetoGb(q resource.Quantity) string {
	return ByteToGb(uint64(q.Value()))
}

func ByteToGb(num uint64) string {
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", f)
}

func ByteToGbiTos(num int64) string {
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", f)
}

func GetFree(t string, u int64) string {
	t1, _ := strconv.ParseFloat(t, 64)
	f1 := float64(u) / (1024 * 1024 * 1024)
	f := t1 - f1
	return fmt.Sprintf("%.2f", f)
}

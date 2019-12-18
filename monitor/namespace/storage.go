package namespace

import (
	"strconv"

	"github.com/zdnscloud/cluster-agent/monitor/event"
	"github.com/zdnscloud/gok8s/client"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	corev1 "k8s.io/api/core/v1"
)

func GetStorage(cli client.Client) map[string]event.StorageSize {
	storages := make(map[string]event.StorageSize)
	storageclusters := storagev1.ClusterList{}
	err := cli.List(ctx, nil, &storageclusters)
	if err != nil {
		return storages
	}
	for _, storagecluster := range storageclusters.Items {
		size := storagecluster.Status.Capacity.Total
		total, _ := strconv.ParseInt(size.Total, 10, 64)
		used, _ := strconv.ParseInt(size.Used, 10, 64)
		storages[storagecluster.Name] = event.StorageSize{
			Total: total,
			Used:  used,
		}
	}
	return storages
}

func getPodsWithPvcs(cli client.Client, namespace string) map[string][]string {
	podsWithPvcs := make(map[string][]string)
	pods := corev1.PodList{}
	err := cli.List(ctx, &client.ListOptions{Namespace: namespace}, &pods)
	if err != nil {
		return podsWithPvcs
	}
	for _, pod := range pods.Items {
		pvcs := make([]string, 0)
		vs := pod.Spec.Volumes
		for _, v := range vs {
			if v.PersistentVolumeClaim == nil {
				continue
			}
			pvcs = append(pvcs, v.PersistentVolumeClaim.ClaimName)
		}
		podsWithPvcs[pod.Name] = pvcs
	}
	return podsWithPvcs
}

func getPvcsWithPv(cli client.Client, namespace string) map[string]string {
	pvcsWithPv := make(map[string]string)
	pvcs := corev1.PersistentVolumeClaimList{}
	err := cli.List(ctx, &client.ListOptions{Namespace: namespace}, &pvcs)
	if err != nil {
		return pvcsWithPv
	}
	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase != "Bound" || pvc.Spec.StorageClassName == nil {
			continue
		}
		pvcsWithPv[pvc.Name] = pvc.Spec.VolumeName
	}
	return pvcsWithPv
}

func getQuotas(cli client.Client, namespace string) (int64, int64, int64) {
	var cpu, mem, storage int64
	resourceQuotas := corev1.ResourceQuotaList{}
	err := cli.List(ctx, &client.ListOptions{Namespace: namespace}, &resourceQuotas)
	if err != nil {
	}
	for _, quota := range resourceQuotas.Items {
		if quota.Spec.Hard != nil {
			sv, ok := quota.Spec.Hard["requests.storage"]
			if ok {
				q := sv.Value()
				if storage == 0 || q < storage {
					storage = q
				}
			}
			cv, ok := quota.Spec.Hard["limits.cpu"]
			if ok {
				q := cv.Value()
				if cpu == 0 || q < cpu {
					cpu = q
				}
			}
			mv, ok := quota.Spec.Hard["limits.memory"]
			if ok {
				q := mv.Value()
				if mem == 0 || q < mem {
					mem = q
				}
			}
		}
	}
	return cpu, mem, storage
}

package servicemesh

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/zdnscloud/cement/slice"
	sm "github.com/zdnscloud/servicemesh"
	pb "github.com/zdnscloud/servicemesh/public"

	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

const (
	StatEndPoint            = "StatSummary"
	DefaultMetricTimeWindow = "1m"
	AllResourceType         = "all"
)

var WorkloadKinds = []string{ResourceTypeDeployment, ResourceTypeDaemonSet, ResourceTypeStatefulSet}

type StatOptions struct {
	ApiServerURL *url.URL
	Namespace    string
	ResourceID   string
	Dsts         []string
	ResourceType string
	ResourceName string
	From         bool
	To           bool
}

func getStat(options *StatOptions) (types.Stat, error) {
	stats, err := getStatsByReq(options.ApiServerURL, buildStatRequest(options), options.ResourceType == ResourceTypePod)
	if err != nil {
		return types.Stat{}, err
	}

	if len(stats) == 0 {
		return types.Stat{}, nil
	}

	return stats[0], nil
}

func getStats(options *StatOptions) (types.Stats, error) {
	return getStatsByReq(options.ApiServerURL, buildStatRequest(options), options.ResourceType == ResourceTypePod)
}

func buildStatRequest(options *StatOptions) *pb.StatSummaryRequest {
	pbResource := &pb.Resource{
		Namespace: options.Namespace,
		Type:      options.ResourceType,
		Name:      options.ResourceName,
	}

	req := &pb.StatSummaryRequest{
		Selector: &pb.ResourceSelection{
			Resource: pbResource,
		},
		TimeWindow: DefaultMetricTimeWindow,
		TcpStats:   true,
		Outbound: &pb.StatSummaryRequest_None{
			None: &pb.Empty{},
		},
	}

	if options.From {
		req.Selector.Resource = &pb.Resource{
			Namespace: options.Namespace,
			Type:      AllResourceType,
		}
		req.Outbound = &pb.StatSummaryRequest_FromResource{FromResource: pbResource}
	} else if options.To {
		req.Selector.Resource = &pb.Resource{
			Namespace: options.Namespace,
			Type:      AllResourceType,
		}
		req.Outbound = &pb.StatSummaryRequest_ToResource{ToResource: pbResource}
	}

	return req
}

func getStatsByReq(apiServerURL *url.URL, req *pb.StatSummaryRequest, isReqPodType bool) (types.Stats, error) {
	var resp pb.StatSummaryResponse
	if err := sm.HandleApiRequest(apiServerURL, StatEndPoint, req, &resp); err != nil {
		return nil, fmt.Errorf("request stats failed: %s", err.Error())
	}

	if e := resp.GetError(); e != nil {
		return nil, fmt.Errorf("stats resp has error: %s", e.Error)
	}

	return pbStatsRespToStats(&resp, isReqPodType), nil
}

func pbStatsRespToStats(resp *pb.StatSummaryResponse, isReqPodType bool) types.Stats {
	stats := make(types.Stats, 0)
	for _, pbStatTable := range resp.GetOk().GetStatTables() {
		for _, pbstat := range pbStatTable.GetPodGroup().GetRows() {
			if isReqPodType {
				if pbstat.Resource.GetType() != ResourceTypePod {
					continue
				}
			} else if slice.SliceIndex(WorkloadKinds, pbstat.Resource.Type) == -1 {
				continue
			}

			stats = append(stats, types.Stat{
				Resource:        pbResourceToResource(pbstat.Resource),
				TimeWindow:      pbstat.TimeWindow,
				Status:          pbstat.Status,
				MeshedPodCount:  int(pbstat.MeshedPodCount),
				RunningPodCount: int(pbstat.RunningPodCount),
				FailedPodCount:  int(pbstat.FailedPodCount),
				BasicStat: types.BasicStat{
					SuccessCount:       int(pbstat.Stats.GetSuccessCount()),
					FailureCount:       int(pbstat.Stats.GetFailureCount()),
					LatencyMsP50:       int(pbstat.Stats.GetLatencyMsP50()),
					LatencyMsP95:       int(pbstat.Stats.GetLatencyMsP95()),
					LatencyMsP99:       int(pbstat.Stats.GetLatencyMsP99()),
					ActualSuccessCount: int(pbstat.Stats.GetActualSuccessCount()),
					ActualFailureCount: int(pbstat.Stats.GetActualFailureCount()),
				},
				TcpStat: types.TcpStat{
					OpenConnections: int(pbstat.TcpStats.GetOpenConnections()),
					ReadBytesTotal:  int(pbstat.TcpStats.GetReadBytesTotal()),
					WriteBytesTotal: int(pbstat.TcpStats.GetWriteBytesTotal()),
				},
				TsStat: types.TrafficSplitStat{
					Apex:   pbstat.TsStats.GetApex(),
					Leaf:   pbstat.TsStats.GetLeaf(),
					Weight: pbstat.TsStats.GetWeight(),
				},
				PodErrors: pbErrorsByPodToPodErrors(pbstat.ErrorsByPod),
			})
		}
	}

	sort.Sort(stats)
	return stats
}

func pbErrorsByPodToPodErrors(pbErrsByPod map[string]*pb.PodErrors) types.PodErrors {
	podErrors := make(types.PodErrors, 0)
	for podName, pbPodErrs := range pbErrsByPod {
		var containerErrs []types.ContainerError
		for _, pbPodErr := range pbPodErrs.GetErrors() {
			containerErrs = append(containerErrs, types.ContainerError{
				Message:   pbPodErr.GetContainer().GetMessage(),
				Container: pbPodErr.GetContainer().GetContainer(),
				Image:     pbPodErr.GetContainer().GetImage(),
				Reason:    pbPodErr.GetContainer().GetReason(),
			})
		}
		podErrors = append(podErrors, types.PodError{
			PodName: podName,
			Errors:  containerErrs,
		})
	}

	sort.Sort(podErrors)
	return podErrors
}

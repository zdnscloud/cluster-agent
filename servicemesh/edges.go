package servicemesh

import (
	"fmt"
	"net/url"
	"sort"

	pb "github.com/zdnscloud/cluster-agent/servicemesh/public"
	"github.com/zdnscloud/cluster-agent/servicemesh/types"
)

const edgeEndPoint = "Edges"

func getEdges(apiServerURL *url.URL, namespace, kind string) (types.Edges, error) {
	var resp pb.EdgesResponse
	if err := apiRequest(apiServerURL, edgeEndPoint, buildEdgesRequest(namespace, kind), &resp); err != nil {
		return nil, fmt.Errorf("request %s edges with namespace %s failed: %s", kind, namespace, err.Error())
	}

	if e := resp.GetError(); e != nil {
		return nil, fmt.Errorf("%s edges response with namespace %s get error: %s", kind, namespace, e.Error)
	}

	return pbEdgesRespToEdges(&resp), nil
}

func buildEdgesRequest(namespace, kind string) *pb.EdgesRequest {
	return &pb.EdgesRequest{
		Selector: &pb.ResourceSelection{
			Resource: &pb.Resource{
				Namespace: namespace,
				Type:      kind,
			},
		},
	}
}

func pbEdgesRespToEdges(resp *pb.EdgesResponse) types.Edges {
	edges := make(types.Edges, 0)
	for _, pbedge := range resp.GetOk().GetEdges() {
		edges = append(edges, &types.Edge{
			Src:      pbResourceToResource(pbedge.Src),
			Dst:      pbResourceToResource(pbedge.Dst),
			ClientID: pbedge.ClientId,
			ServerID: pbedge.ServerId,
			Msg:      pbedge.NoIdentityMsg,
		})
	}

	sort.Sort(edges)
	return edges
}

func pbResourceToResource(pbResource *pb.Resource) types.Resource {
	return types.Resource{
		Namespace: pbResource.GetNamespace(),
		Type:      pbResource.GetType(),
		Name:      pbResource.GetName(),
	}
}

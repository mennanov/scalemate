//revive:disable
package frontend

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
)

func (s *SchedulerFrontend) GetResourceRequest(context.Context, *scheduler_proto.ResourceRequestLookupRequest) (*scheduler_proto.ResourceRequest, error) {
	panic("implement me")
}

func (s *SchedulerFrontend) ListResourceRequests(context.Context, *scheduler_proto.ListResourceRequestsRequest) (*scheduler_proto.ListResourceRequestsResponse, error) {
	panic("implement me")
}

func (s *SchedulerFrontend) GetNode(context.Context, *scheduler_proto.NodeLookupRequest) (*scheduler_proto.Node, error) {
	panic("implement me")
}

func (s *SchedulerFrontend) ListNodes(context.Context, *scheduler_proto.ListNodesRequest) (*scheduler_proto.ListNodesResponse, error) {
	panic("implement me")
}

func (s *SchedulerFrontend) ListNodeLabels(context.Context, *types.Empty) (*scheduler_proto.ListNodeLabelsResponse, error) {
	panic("implement me")
}

package server

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
)

func (s *SchedulerServer) ReceiveContainerUpdates(*scheduler_proto.ContainerLookupRequest, scheduler_proto.Scheduler_ReceiveContainerUpdatesServer) error {
	panic("implement me")
}

func (s *SchedulerServer) GetResourceRequest(context.Context, *scheduler_proto.ResourceRequestLookupRequest) (*scheduler_proto.ResourceRequest, error) {
	panic("implement me")
}

func (s *SchedulerServer) ListResourceRequests(context.Context, *scheduler_proto.ListResourceRequestsRequest) (*scheduler_proto.ListResourceRequestsResponse, error) {
	panic("implement me")
}

func (s *SchedulerServer) GetNode(context.Context, *scheduler_proto.NodeLookupRequest) (*scheduler_proto.Node, error) {
	panic("implement me")
}

func (s *SchedulerServer) ListNodes(context.Context, *scheduler_proto.ListNodesRequest) (*scheduler_proto.ListNodesResponse, error) {
	panic("implement me")
}

func (s *SchedulerServer) ListNodeLabels(context.Context, *types.Empty) (*scheduler_proto.ListNodeLabelsResponse, error) {
	panic("implement me")
}

package server

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
)

// ListNodes lists Nodes that satisfy the given criteria.
func (s SchedulerServer) ListNodes(
	ctx context.Context,
	r *scheduler_proto.ListNodesRequest,
) (*scheduler_proto.ListNodesResponse, error) {
	var nodes models.Nodes
	totalCount, err := nodes.List(s.db, r)
	if err != nil {
		return nil, err
	}

	response := &scheduler_proto.ListNodesResponse{TotalCount: totalCount}
	for _, node := range nodes {
		nodeProto, err := node.ToProto(nil)
		if err != nil {
			return nil, errors.Wrap(err, "node.ToProto failed")
		}
		response.Nodes = append(response.Nodes, nodeProto)
	}

	return response, nil
}

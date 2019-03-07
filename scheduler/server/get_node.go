package server

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

// GetNode gets the Node by its ID.
func (s SchedulerServer) GetNode(
	ctx context.Context,
	r *scheduler_proto.NodeLookupRequest,
) (*scheduler_proto.Node, error) {
	logger := ctxlogrus.Extract(ctx)

	node := &models.Node{}
	node.ID = r.NodeId

	if err := node.LoadFromDB(s.db); err != nil {
		return nil, err
	}

	jobProto, err := node.ToProto(nil)
	if err != nil {
		logger.WithError(err).WithField("node", node).Errorf("node.ToProto failed")
	}
	return jobProto, nil
}

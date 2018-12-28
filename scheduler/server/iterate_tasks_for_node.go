package server

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
)

// IterateTasksForNode marks the Node as ONLINE and starts sending new Tasks for it to run.
// Once the RPC is done the Node is marked as OFFLINE.
func (s SchedulerServer) IterateTasksForNode(
	_ *empty.Empty,
	stream scheduler_proto.Scheduler_IterateTasksForNodeServer,
) error {
	ctx := stream.Context()
	logger := ctxlogrus.Extract(ctx)
	// Authorize this request first.
	ctxClaims := ctx.Value(auth.ContextKeyClaims)
	if ctxClaims == nil {
		return status.Error(codes.Unauthenticated, "no JWT claims found")
	}

	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}
	if claims.NodeName == "" {
		return status.Error(
			codes.PermissionDenied, "node name is missing in JWT. Authenticate as a Node, not as a client")
	}

	// Lookup the Node in DB.
	node := &models.Node{}
	if err := node.Get(s.DB, claims.Username, claims.NodeName); err != nil {
		logger.WithFields(logrus.Fields{
			"nodeUsername": claims.Username,
			"nodeName":     claims.NodeName,
		}).Warning("Could not find a Node in DB in IterateTasksForNode method. Possible data inconsistency!")
		return status.Errorf(codes.NotFound, "node with name '%s' ")
	}

	if node.Status == models.Enum(scheduler_proto.Node_STATUS_ONLINE) {
		return status.Error(codes.FailedPrecondition, "this Node is already online (receiving Tasks)")
	}

	tx := s.DB.Begin()
	connectEvent, err := s.ConnectNode(tx, node)
	if err != nil {
		return errors.Wrap(err, "failed to ConnectNode")
	}
	if err := events.CommitAndPublish(tx, s.Producer, connectEvent); err != nil {
		return errors.Wrap(err, "events.CommitAndPublish failed")
	}

	// Set the Node's status as offline when this RPC exits.
	defer func() {
		tx := s.DB.Begin()
		disconnectEvent, err := s.DisconnectNode(tx, node)
		if err != nil {
			logger.WithError(err).Error("failed to DisconnectNode")
		}
		if err := events.CommitAndPublish(tx, s.Producer, disconnectEvent); err != nil {
			logger.WithError(err).Error("events.CommitAndPublish failed")
		}
	}()

	for {
		select {
		case taskProto, ok := <-s.NewTasksByNodeID[node.ID]:
			if !ok {
				// Channel is closed. Node is probably shutting down.
				return nil
			}
			if err := stream.Send(taskProto); err != nil {
				return errors.Wrap(err, "failed to send a Task to the Tasks stream")
			}

		case <-ctx.Done():
			return nil
		}
	}
}

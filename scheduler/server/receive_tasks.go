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
	"github.com/mennanov/scalemate/shared/utils"
)

func (s SchedulerServer) ReceiveTasks(_ *empty.Empty, stream scheduler_proto.Scheduler_ReceiveTasksServer) error {
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
		}).Warning("Could not find a Node in DB in ReceiveTasks method. Possible data inconsistency!")
		return status.Errorf(codes.NotFound, "node with name '%s' ")
	}

	if node.Status == models.Enum(scheduler_proto.Node_STATUS_ONLINE) {
		return status.Error(codes.FailedPrecondition, "this Node is already receiving Tasks")
	}

	publisher, err := events.NewAMQPPublisher(s.AMQPConnection, utils.SchedulerAMQPExchangeName)
	if err != nil {
		return errors.Wrap(err, "failed to create NewAMQPPublisher")
	}

	if err := s.ConnectNode(ctx, node, publisher); err != nil {
		return errors.Wrap(err, "failed to ConnectNode")
	}
	// Set the Node's status as offline when this RPC exits.
	defer s.DisconnectNode(ctx, node, publisher)

	for {
		select {
		case taskProto := <-s.ConnectedNodes[node.ID]:
			if err := stream.Send(taskProto); err != nil {
				return errors.Wrap(err, "failed to send a Task to the Tasks stream")
			}

		case <-ctx.Done():
			return nil
		}
	}
}

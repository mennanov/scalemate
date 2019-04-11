package server

import (
	"time"

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
	if err := node.Get(s.db, claims.Username, claims.NodeName); err != nil {
		logger.WithError(err).WithFields(logrus.Fields{
			"nodeUsername": claims.Username,
			"nodeName":     claims.NodeName,
		}).Warning("Could not find a Node in DB in IterateTasksForNode method. Possible data inconsistency!")
		return status.Errorf(codes.NotFound, "node with name '%s' ")
	}

	if node.Status == utils.Enum(scheduler_proto.Node_STATUS_ONLINE) {
		return status.Error(codes.FailedPrecondition, "this Node is already online (receiving Tasks)")
	}

	// Mark the Node as ONLINE.
	errCh := make(chan error)
	go func(e chan error) {
		tx := s.db.Begin()
		// Lock tasksForNodesMux until this function exits due to a possible race condition when the same Node connects
		// multiple times simultaneously.
		s.tasksForNodesMux.Lock()
		defer s.tasksForNodesMux.Unlock()

		_, ok = s.tasksForNodes[node.ID]
		if ok {
			e <- ErrNodeAlreadyConnected
			return
		}
		now := time.Now()
		connectEvent, err := node.Updates(tx, map[string]interface{}{
			"connected_at": &now,
			"status":       utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		})
		if err != nil {
			e <- errors.Wrap(err, "node.Updates failed")
			return
		}
		s.tasksForNodes[node.ID] = make(chan *scheduler_proto.Task)

		if err != nil {
			e <- utils.RollbackTransaction(tx, errors.Wrap(err, "failed to connectNode"))
			return
		}
		if err := events.CommitAndPublish(s.producer, connectEvent); err != nil {
			e <- errors.Wrap(err, "events.CommitAndPublish failed")
			return
		}
		e <- nil
	}(errCh)

	err := <-errCh
	if err != nil {
		return err
	}

	s.tasksForNodesMux.RLock()
	tasks, ok := s.tasksForNodes[node.ID]
	s.tasksForNodesMux.RUnlock()

	// Set the Node's status as OFFLINE when this RPC exits.
	defer func(tasks chan *scheduler_proto.Task) {
		tx := s.db.Begin()

		if !ok {
			logger.Errorf("tasks channel not found for Node ID %d", node.ID)
			return
		}
		now := time.Now()
		nodeUpdatedEvent, err := node.Updates(tx, map[string]interface{}{
			"disconnected_at": &now,
			"status":          utils.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		})
		if err != nil {
			logger.WithError(utils.RollbackTransaction(tx, err)).Error("node.Updates failed")
			return
		}

		close(tasks)
		s.tasksForNodesMux.Lock()
		delete(s.tasksForNodes, node.ID)
		s.tasksForNodesMux.Unlock()

		if err := events.CommitAndPublish(s.producer, nodeUpdatedEvent); err != nil {
			logger.WithError(err).Error("events.CommitAndPublish failed")
		}
	}(tasks)

	if !ok {
		return status.Errorf(codes.Internal, "tasks channel not found in tasksForNodes for Node %d", node.ID)
	}

	for {
		select {
		case taskProto, ok := <-tasks:
			if !ok {
				// Channel is closed. This should never happen. Node was disconnected prematurely.
				logger.WithFields(logrus.Fields{
					"nodeUsername": node.Username,
					"nodeName":     node.Name,
				}).Warning("Tasks channel is unexpectedly closed")
				return status.Error(codes.Internal, "tasks channel is unexpectedly closed")
			}
			if err := stream.Send(taskProto); err != nil {
				return errors.Wrap(err, "failed to send a Task to the Tasks stream")
			}

		case <-ctx.Done():
			return ctx.Err()

		case <-s.gracefulStop:
			return status.Error(codes.Unavailable, "service is shutting down")
		}
	}
}

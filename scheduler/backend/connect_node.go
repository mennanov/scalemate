package backend

import (
	"database/sql"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *SchedulerBackend) ConnectNode(request *scheduler_proto.ConnectNodeRequest, srv scheduler_proto.SchedulerBackEnd_ConnectNodeServer) error {
	ctx := srv.Context()
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get auth claims")
	}

	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return errors.Wrap(err, "failed to start a transaction")
	}

	node, err := models.NewNodeFromDB(tx, "username = ? AND name = ?", claims.Username, claims.NodeName)
	if err != nil {
		return errors.Wrap(err, "failed to get the node by username and name")
	}

	if node.Status != scheduler_proto.Node_OFFLINE {
		return utils.RollbackTransaction(tx, status.Error(codes.FailedPrecondition, "node is already connected"))
	}

	defer func() {
		// Mark the Node as OFFLINE when the outer function returns.
		tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			s.logger.WithError(err).Error("failed to start transaction")
			return
		}
		nodeDisconnectedUpdates := map[string]interface{}{
			"status": scheduler_proto.Node_OFFLINE,
		}
		if err := node.Update(tx, nodeDisconnectedUpdates); err != nil {
			s.logger.WithError(err).Error("failed to update node (mark as disconnected)")
			return
		}

		nodeProto, nodeMask, err := node.ToProtoFromUpdates(nodeDisconnectedUpdates)
		if err != nil {
			s.logger.WithError(utils.RollbackTransaction(tx, errors.WithStack(err))).Error("node.ToProtoFromUpdates failed")
			return
		}

		if err := events.CommitAndPublish(tx, s.producer, &events_proto.Event{
			Payload: &events_proto.Event_SchedulerNodeDisconnected{
				SchedulerNodeDisconnected: &scheduler_proto.NodeDisconnectedEvent{
					Node:     nodeProto,
					NodeMask: nodeMask,
				},
			},
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			s.logger.WithError(err).Error("failed to CommitAndPublish NodeDisconnectedEvent")
		}
	}()

	nodeConnectedUpdates := map[string]interface{}{
		"status":                   scheduler_proto.Node_ONLINE,
		"cpu_available":            request.CpuAvailable,
		"gpu_available":            request.GpuAvailable,
		"memory_available":         request.MemoryAvailable,
		"disk_available":           request.DiskAvailable,
		"network_ingress_capacity": request.NetworkIngressCapacity,
		"network_egress_capacity":  request.NetworkEgressCapacity,
	}
	if err := node.Update(tx, nodeConnectedUpdates); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "failed to update the node"))
	}

	nodeProto, nodeMask, err := node.ToProtoFromUpdates(nodeConnectedUpdates)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	if err := events.CommitAndPublish(tx, s.producer, &events_proto.Event{
		Payload: &events_proto.Event_SchedulerNodeConnected{
			SchedulerNodeConnected: &scheduler_proto.NodeConnectedEvent{
				Node:     nodeProto,
				NodeMask: nodeMask,
			},
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

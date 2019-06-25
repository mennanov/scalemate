package server

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// StopContainer updates the Container's status to CANCELLED.
func (s *SchedulerServer) StopContainer(ctx context.Context, request *scheduler_proto.ContainerLookupRequest) (*types.Empty, error) {
	return s.updateContainerStatus(ctx, request, scheduler_proto.Container_STOPPED)
}

func (s *SchedulerServer) updateContainerStatus(
	ctx context.Context,
	request *scheduler_proto.ContainerLookupRequest,
	toStatus scheduler_proto.Container_Status,
) (*types.Empty, error) {
	container, err := models.NewContainerFromDB(s.db, request.ContainerId, false)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := utils.ClaimsUsernameEqual(ctx, container.Username); err != nil {
		return nil, errors.WithStack(err)
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}
	if err := container.ValidateNewStatus(toStatus); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	containerUpdateEvent, err := container.Update(tx, map[string]interface{}{"status": toStatus})
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	if err := events.CommitAndPublish(tx, s.producer, containerUpdateEvent); err != nil {
		return nil, errors.WithStack(err)
	}
	return &types.Empty{}, nil
}

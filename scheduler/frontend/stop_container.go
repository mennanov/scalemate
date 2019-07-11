package frontend

import (
	"context"
	"database/sql"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// StopContainer updates the Container's status to CANCELLED.
func (s *SchedulerFrontend) StopContainer(ctx context.Context, request *scheduler_proto.ContainerLookupRequest) (*types.Empty, error) {
	return s.updateContainerStatus(ctx, request, scheduler_proto.Container_STOPPED)
}

func (s *SchedulerFrontend) updateContainerStatus(
	ctx context.Context,
	request *scheduler_proto.ContainerLookupRequest,
	toStatus scheduler_proto.Container_Status,
) (*types.Empty, error) {
	container, err := models.NewContainerFromDB(s.db, request.ContainerId)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := utils.ClaimsUsernameEqual(ctx, container.Username); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := container.ValidateNewStatus(toStatus); err != nil {
		return nil, errors.WithStack(err)
	}

	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}

	containerUpdates := map[string]interface{}{"status": toStatus}
	if err := container.Update(tx, containerUpdates); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	containerProto, containerMask, err := container.ToProtoFromUpdates(containerUpdates)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	if err := events.CommitAndPublish(tx, s.producer, &events_proto.Event{
		Payload: &events_proto.Event_SchedulerContainerStopped{
			SchedulerContainerStopped: &scheduler_proto.ContainerStoppedEvent{
				Container:     containerProto,
				ContainerMask: containerMask,
			},
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, errors.WithStack(err)
	}
	return &types.Empty{}, nil
}

package frontend

import (
	"context"
	"database/sql"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// CreateContainer creates a new Container with a corresponding CurrentResourceRequest.
func (s SchedulerFrontend) CreateContainer(
	ctx context.Context,
	r *scheduler_proto.ContainerWithResourceRequest,
) (*scheduler_proto.ContainerWithResourceRequest, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := ValidateContainerFields(r.Container); err != nil {
		return nil, err
	}
	if err := ValidateResourceRequestFields(r.ResourceRequest, nil); err != nil {
		return nil, err
	}
	// Set the Container's Username from claims.
	r.Container.Username = claims.Username
	// Create a Container.
	container := models.NewContainerFromProto(r.Container)
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}

	if err := container.Create(tx); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to create a Container"))
	}
	// Create a corresponding CurrentResourceRequest.
	request := models.NewResourceRequestFromProto(r.ResourceRequest)
	request.ContainerId = container.Id

	if err := request.Create(tx); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to create a CurrentResourceRequest"))
	}

	if err := events.CommitAndPublish(tx, s.producer, &events_proto.Event{
		Payload: &events_proto.Event_SchedulerContainerCreated{
			SchedulerContainerCreated: &scheduler_proto.ContainerCreatedEvent{
				Container:       &container.Container,
				ResourceRequest: &request.ResourceRequest,
			},
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, errors.WithStack(err)
	}

	return &scheduler_proto.ContainerWithResourceRequest{
		Container:       &container.Container,
		ResourceRequest: &request.ResourceRequest,
	}, nil
}

package server

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// CreateContainer creates a new Container with a corresponding ResourceRequest.
func (s SchedulerServer) CreateContainer(
	ctx context.Context,
	r *scheduler_proto.ContainerWithResourceRequest,
) (*scheduler_proto.ContainerWithResourceRequest, error) {
	if err := ValidateContainerFields(r.Container); err != nil {
		return nil, err
	}
	if err := utils.ClaimsUsernameEqual(ctx, r.Container.Username); err != nil {
		return nil, err
	}
	if err := ValidateResourceRequestFields(r.ResourceRequest, nil); err != nil {
		return nil, err
	}
	// Create a Container.
	container := models.NewContainerFromProto(r.Container)
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}
	err := container.Create(tx)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to create a Container"))
	}
	// Create a corresponding ResourceRequest.
	request := models.NewResourceRequestFromProto(r.ResourceRequest)
	request.ContainerId = container.Id
	requestEvent, err := request.Create(tx)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to create a ResourceRequest"))
	}

	if err := events.CommitAndPublish(tx, s.producer, containerEvent, requestEvent); err != nil {
		return nil, errors.WithStack(err)
	}

	return &scheduler_proto.ContainerWithResourceRequest{
		Container:       &container.Container,
		ResourceRequest: &request.ResourceRequest,
	}, nil
}

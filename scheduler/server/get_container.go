package server

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// GetContainer gets the Container by its ID. Container can be accessed by its owner only.
func (s SchedulerServer) GetContainer(
	ctx context.Context,
	r *scheduler_proto.ContainerLookupRequest,
) (*scheduler_proto.Container, error) {
	container, err := models.NewContainerFromDB(s.db, r.ContainerId, false)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := utils.ClaimsUsernameEqual(ctx, container.Username); err != nil {
		return nil, errors.WithStack(err)
	}
	return &container.Container, nil
}

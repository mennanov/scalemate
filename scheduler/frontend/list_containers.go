package frontend

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
)

// ListContainers lists existing Containers.
func (s *SchedulerFrontend) ListContainers(
	ctx context.Context, request *scheduler_proto.ListContainersRequest,
) (*scheduler_proto.ListContainersResponse, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := validation.ValidateStruct(request,
		validation.Field(&request.Limit, validation.Required),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	containers, count, err := models.ListContainers(s.db, claims.Username, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find containers")
	}
	containersForResponse := make([]*scheduler_proto.Container, len(containers))
	for i, container := range containers {
		containersForResponse[i] = &container.Container
	}

	response := &scheduler_proto.ListContainersResponse{
		Containers: containersForResponse,
		TotalCount: count,
	}
	return response, nil
}

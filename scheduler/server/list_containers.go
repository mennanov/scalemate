package server

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *SchedulerServer) ListContainers(
	ctx context.Context, request *scheduler_proto.ListContainersRequest,
) (*scheduler_proto.ListContainersResponse, error) {
	if err := validation.ValidateStruct(request,
		validation.Field(&request.Username, validation.Required),
		validation.Field(&request.Limit, validation.Required),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := utils.ClaimsUsernameEqual(ctx, request.Username); err != nil {
		return nil, errors.WithStack(err)
	}
	containers, count, err := models.ListContainers(s.db, request)
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

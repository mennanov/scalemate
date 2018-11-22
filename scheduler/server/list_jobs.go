package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListJobs lists Jobs owned by the User.
func (s SchedulerServer) ListJobs(ctx context.Context, r *scheduler_proto.ListJobsRequest) (*scheduler_proto.ListJobsResponse, error) {
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	if claims.Username != r.Username && claims.Role != accounts_proto.User_ADMIN {
		return nil, status.Errorf(codes.PermissionDenied, "you can only view Jobs for user '%s'", claims.Username)
	}

	var jobs models.Jobs
	totalCount, err := jobs.List(s.DB, r)
	if err != nil {
		return nil, err
	}

	response := &scheduler_proto.ListJobsResponse{TotalCount: totalCount}
	for _, job := range jobs {
		jobProto, err := job.ToProto(nil)
		if err != nil {
			return nil, errors.Wrap(err, "job.ToProto failed")
		}
		response.Jobs = append(response.Jobs, jobProto)
	}

	return response, nil
}

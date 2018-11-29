package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
)

// ListTasks lists Tasks owned by the User.
func (s SchedulerServer) ListTasks(ctx context.Context, r *scheduler_proto.ListTasksRequest) (*scheduler_proto.ListTasksResponse, error) {
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	if claims.Username != r.Username && claims.Role != accounts_proto.User_ADMIN {
		return nil, status.Errorf(codes.PermissionDenied, "you can only view Tasks for user '%s'", claims.Username)
	}

	var tasks models.Tasks
	totalCount, err := tasks.List(s.DB, r)
	if err != nil {
		return nil, err
	}

	response := &scheduler_proto.ListTasksResponse{TotalCount: totalCount}
	for _, task := range tasks {
		taskProto, err := task.ToProto(nil)
		if err != nil {
			return nil, errors.Wrap(err, "task.ToProto failed")
		}
		response.Tasks = append(response.Tasks, taskProto)
	}

	return response, nil
}

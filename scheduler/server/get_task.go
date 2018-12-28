package server

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
)

// GetTask gets the Task by its ID. Task can be accessed by its owner (or admin) only.
func (s SchedulerServer) GetTask(
	ctx context.Context,
	r *scheduler_proto.GetTaskRequest,
) (*scheduler_proto.Task, error) {
	logger := ctxlogrus.Extract(ctx)
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	task := &models.Task{}
	task.ID = r.TaskId

	if err := task.LoadFromDB(s.DB); err != nil {
		return nil, err
	}

	if err := task.LoadJobFromDB(s.DB, "username"); err != nil {
		return nil, err
	}

	if task.Job.Username != claims.Username && claims.Role != accounts_proto.User_ADMIN {
		logger.WithFields(logrus.Fields{
			"task":    task,
			"request": r,
			"claims":  claims,
		}).Warn("permission denied in GetTask")
		return nil, status.Error(codes.PermissionDenied, "the Task requested is not owned by you")
	}

	taskProto, err := task.ToProto(nil)
	if err != nil {
		logger.WithError(err).WithField("task", task).Errorf("task.ToProto failed")
	}
	return taskProto, nil
}

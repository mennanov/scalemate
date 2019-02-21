package server

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// CancelTask cancels the currently running Task.
func (s SchedulerServer) CancelTask(
	ctx context.Context,
	r *scheduler_proto.TaskLookupRequest,
) (*scheduler_proto.Task, error) {
	logger := ctxlogrus.Extract(ctx)
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	task := &models.Task{}
	task.ID = r.TaskId

	if err := task.LoadFromDB(s.db); err != nil {
		return nil, err
	}

	if err := task.LoadJobFromDB(s.db); err != nil {
		return nil, errors.Wrap(err, "task.LoadJobFromDB failed")
	}

	if task.Job.Username != claims.Username && claims.Role != accounts_proto.User_ADMIN {
		logger.WithFields(logrus.Fields{
			"task":    task,
			"job":     task.Job,
			"request": r,
			"claims":  claims,
		}).Warn("permission denied in CancelTask")
		return nil, status.Error(
			codes.PermissionDenied, "corresponding Job username does not match currently authenticated user")
	}

	tx := s.db.Begin()
	taskStatusEvent, err := task.UpdateStatus(tx, scheduler_proto.Task_STATUS_CANCELLED)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "task.UpdateStatus failed"))
	}

	if err := events.CommitAndPublish(tx, s.producer, taskStatusEvent); err != nil {
		return nil, errors.Wrap(err, "failed to send and commit events")
	}

	taskProto, err := task.ToProto(nil)
	if err != nil {
		logger.WithError(err).WithField("task", task).Errorf("task.ToProto failed")
	}
	return taskProto, nil
}

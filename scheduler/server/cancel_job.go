package server

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// CancelJob cancels the currently running Job. Also cancels the corresponding running Tasks.
func (s SchedulerServer) CancelJob(
	ctx context.Context,
	r *scheduler_proto.JobLookupRequest,
) (*scheduler_proto.Job, error) {
	logger := ctxlogrus.Extract(ctx)
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	job := &models.Job{}
	job.ID = r.JobId

	if err := job.LoadFromDB(s.db); err != nil {
		return nil, err
	}

	if job.Username != claims.Username && claims.Role != accounts_proto.User_ADMIN {
		logger.WithFields(logrus.Fields{
			"job":     job,
			"request": r,
			"claims":  claims,
		}).Warn("permission denied in CancelJob")
		return nil, status.Error(codes.PermissionDenied, "Job username does not match currently authenticated user")
	}

	tx := s.db.Begin()
	var updateEvents []*events_proto.Event
	jobStatusEvent, err := job.UpdateStatus(tx, scheduler_proto.Job_STATUS_CANCELLED)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "job.UpdateStatus failed"))
	}
	updateEvents = append(updateEvents, jobStatusEvent)

	if err := job.LoadTasksFromDB(tx, "id", "status"); err != nil {
		return nil, errors.Wrap(err, "job.LoadTasksFromDB failed")
	}

	for _, task := range job.Tasks {
		if task.IsTerminated() {
			// Skip terminated Tasks.
			continue
		}
		taskUpdatedEvent, err := task.UpdateStatus(tx, scheduler_proto.Task_STATUS_CANCELLED)
		if err != nil {
			return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "task.UpdateStatus failed"))
		}
		updateEvents = append(updateEvents, taskUpdatedEvent)
	}

	if err := events.CommitAndPublish(s.producer, updateEvents); err != nil {
		return nil, errors.Wrap(err, "failed to send and commit events")
	}

	jobProto, err := job.ToProto(nil)
	if err != nil {
		logger.WithError(err).WithField("job", job).Errorf("job.ToProto failed")
	}
	return jobProto, nil
}

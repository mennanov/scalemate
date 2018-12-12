package server

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
)

// CreateJob creates a new Job.
func (s SchedulerServer) CreateJob(ctx context.Context, r *scheduler_proto.Job) (*scheduler_proto.Job, error) {
	logger := ctxlogrus.Extract(ctx)
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}
	if claims.Username != r.Username {
		msg := "Job username does not match currently authenticated user"
		logger.WithFields(logrus.Fields{
			"job":    r,
			"claims": claims,
		}).Warn(msg)
		return nil, status.Error(codes.PermissionDenied, msg)
	}
	if r.Id != 0 {
		return nil, status.Error(codes.InvalidArgument, "field Id is readonly")
	}
	if r.Status != 0 {
		return nil, status.Error(codes.InvalidArgument, "field Status is readonly")
	}
	if r.CreatedAt != nil {
		return nil, status.Error(codes.InvalidArgument, "field CreatedAt is readonly")
	}
	if r.UpdatedAt != nil {
		return nil, status.Error(codes.InvalidArgument, "field UpdatedAt is readonly")
	}

	job := &models.Job{}
	if err := job.FromProto(r); err != nil {
		return nil, errors.Wrap(err, "job.FromProto failed")
	}
	// Set the initial Job status.
	job.Status = models.Enum(scheduler_proto.Job_STATUS_PENDING)

	tx := s.DB.Begin()
	event, err := job.Create(tx)
	if err != nil {
		return nil, errors.Wrap(err, "job.Create failed")
	}
	if err := events.CommitAndPublish(tx, s.Producer, event); err != nil {
		return nil, errors.Wrap(err, "failed to send and commit events")
	}
	jobProto, err := job.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "job.ToProto")
	}
	return jobProto, nil
}

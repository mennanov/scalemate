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

// GetJob gets the Job by its ID. Job can be accessed by its owner (or admin) only.
func (s SchedulerServer) GetJob(
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
		}).Warn("permission denied in GetJob")
		return nil, status.Error(codes.PermissionDenied, "Job username does not match currently authenticated user")
	}

	jobProto, err := job.ToProto(nil)
	if err != nil {
		logger.WithError(err).WithField("job", job).Errorf("job.ToProto failed")
	}
	return jobProto, nil
}

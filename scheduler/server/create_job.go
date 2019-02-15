package server

import (
	"context"
	"path/filepath"
	"strings"

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
	if r.RunConfig == nil {
		return nil, status.Error(codes.InvalidArgument, "RunConfig can not be nil")
	}
	for localPort, remotePort := range r.RunConfig.Ports {
		if err := validatePort(localPort); err != nil {
			return nil, err
		}
		if err := validatePort(remotePort); err != nil {
			return nil, err
		}
	}
	if err := validatePaths(r.RunConfig.Volumes, func(left, right string) string { return left }); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid volume paths: %s", err.Error())
	}
	if err := validatePaths(r.RunConfig.DownloadPaths, func(left, right string) string { return left }); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid download paths: %s", err.Error())
	}
	if err := validatePaths(r.RunConfig.UploadPaths, func(left, right string) string { return right }); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid upload paths: %s", err.Error())
	}

	job := &models.Job{}
	if err := job.FromProto(r); err != nil {
		return nil, errors.Wrap(err, "job.FromProto failed")
	}

	tx := s.db.Begin()
	event, err := job.Create(tx)
	if err != nil {
		return nil, errors.Wrap(err, "job.Create failed")
	}
	if err := events.CommitAndPublish(tx, s.producer, event); err != nil {
		return nil, errors.Wrap(err, "failed to send and commit events")
	}
	jobProto, err := job.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "job.ToProto")
	}
	return jobProto, nil
}

func validatePort(p uint32) error {
	if p < 0 || p > 65535 {
		return status.Errorf(codes.InvalidArgument, "network port is out of range: %d", p)
	}
	return nil
}

func validateRelativePath(path string) error {
	if filepath.IsAbs(path) {
		return status.Errorf(codes.InvalidArgument, "absolute path is used as relative: %s", path)
	}
	basePath := "/base/"
	relPath, err := filepath.Rel(basePath, filepath.Join(basePath, path))
	if err != nil {
		return errors.Errorf("invalid relative path: %s: %s", path, err.Error())
	}
	if strings.HasPrefix(relPath, "../") {
		return errors.Errorf("invalid relative path: %s", path)
	}
	return nil
}

func validatePaths(paths map[string]string, relPathGetter func(s1, s2 string) string) error {
	for left, right := range paths {
		if left == "" {
			return errors.New("left side path is empty")
		}
		if right == "" {
			return errors.New("right side path is empty")
		}
		if err := validateRelativePath(relPathGetter(left, right)); err != nil {
			return err
		}
	}
	return nil
}

package server

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
)

// IterateTasks sends a stream of Tasks for a given Job.
func (s SchedulerServer) IterateTasks(
	req *scheduler_proto.IterateTasksRequest,
	stream scheduler_proto.Scheduler_IterateTasksServer,
) error {
	ctx := stream.Context()
	logger := ctxlogrus.Extract(ctx)
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}
	job := &models.Job{}
	job.ID = req.JobId
	if err := job.LoadFromDB(s.db); err != nil {
		return err
	}

	if job.Username != claims.Username {
		logger.WithFields(logrus.Fields{
			"job":     job,
			"request": req,
			"claims":  claims,
		}).Warn("permission denied in IterateTasks")
		return status.Error(codes.PermissionDenied, "Job username does not match currently authenticated user")
	}

	if req.IncludeExisting {
		// Stream existing Tasks first.
		if err := job.LoadTasksFromDB(s.db); err == nil {
			for _, task := range job.Tasks {
				taskProto, err := task.ToProto(nil)
				if err != nil {
					return errors.Wrap(err, "task.ToProto failed")
				}
				if err := stream.Send(taskProto); err != nil {
					return errors.Wrap(err, "failed to stream a Task")
				}
			}
		}
	}

	if job.HasTerminated() {
		// Job has terminated. No new Tasks are possible.
		return nil
	}

	// Create a Task channel to receive newly created Tasks sent by event listeners.
	s.NewTasksByJobID[job.ID] = make(chan *scheduler_proto.Task)
	// Delete this channel once iteration is over.
	defer delete(s.NewTasksByJobID, job.ID)

	for {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "Tasks iteration is cancelled by client")
		case taskProto, ok := <-s.NewTasksByJobID[job.ID]:
			if !ok {
				// Channel is closed. No new Tasks are expected to be created: iteration is finished.
				return nil
			}
			if err := stream.Send(taskProto); err != nil {
				return errors.Wrap(err, "failed to stream a Task")
			}
		case <-s.gracefulStop:
			return status.Error(codes.Unavailable, "service is shutting down")
		}
	}
}

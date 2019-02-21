package server

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// HandleJobPending schedules the pending Job.
func (s *SchedulerServer) HandleJobPending(eventProto *events_proto.Event) error {
	eventPayload, err := events.NewModelProtoFromEvent(eventProto)
	if err != nil {
		return errors.Wrap(err, "events.NewModelProtoFromEvent failed")
	}
	jobProto, ok := eventPayload.(*scheduler_proto.Job)
	if !ok {
		return errors.Wrap(err, "failed to convert message event proto to *scheduler_proto.Job")
	}
	if jobProto.Status != scheduler_proto.Job_STATUS_PENDING {
		return nil
	}

	job := &models.Job{}
	if err := job.FromProto(jobProto); err != nil {
		return errors.Wrap(err, "job.FromProto failed")
	}
	if err := job.LoadFromDB(s.db); err != nil {
		return errors.Wrap(err, "job.LoadFromDB failed")
	}
	tx := s.db.Begin()
	node, err := job.FindSuitableNode(tx)
	if err != nil {
		wrappedErr := errors.Wrap(err, "job.FindSuitableNode failed")
		if st, ok := status.FromError(errors.Cause(err)); ok {
			if st.Code() == codes.NotFound {
				s.logger.WithField("job", job).Info("no suitable Node could be found for a new Job")
				wrappedErr = nil
			}
		}
		return utils.RollbackTransaction(tx, wrappedErr)
	}
	schedulingEvents, err := job.CreateTask(tx, node)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "job.CreateTask failed"))
	}
	if err := events.CommitAndPublish(tx, s.producer, schedulingEvents...); err != nil {
		return errors.Wrap(err, "events.CommitAndPublish failed")
	}
	return nil
}

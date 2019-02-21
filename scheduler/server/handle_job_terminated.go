package server

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
)

// HandleJobTerminated closes the corresponding Tasks channel when the Job is terminated.
func (s *SchedulerServer) HandleJobTerminated(eventProto *events_proto.Event) error {
	eventPayload, err := events.NewModelProtoFromEvent(eventProto)
	if err != nil {
		return errors.Wrap(err, "events.NewModelProtoFromEvent failed")
	}
	jobProto, ok := eventPayload.(*scheduler_proto.Job)
	if !ok {
		return errors.Wrap(err, "failed to convert message event proto to *scheduler_proto.Job")
	}
	job := &models.Job{}
	if err := job.FromProto(jobProto); err != nil {
		return errors.Wrap(err, "job.FromProto failed")
	}
	// Verify that the Job has terminated.
	if !job.IsTerminated() {
		return nil
	}

	if ch, ok := s.NewTasksByJobID[job.ID]; ok {
		// Close the corresponding Tasks channel as there can't be any future Tasks for this terminated Job.
		close(ch)
	}
	return nil
}

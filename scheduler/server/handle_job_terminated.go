package server

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
)

// HandleJobTerminated closes the corresponding Tasks channel when the Container is terminated.
func (s *SchedulerServer) HandleJobTerminated(eventProto *events_proto.Event) error {
	eventPayload, err := events.NewModelProtoFromEvent(eventProto)
	if err != nil {
		return errors.Wrap(err, "events.NewModelProtoFromEvent failed")
	}
	jobProto, ok := eventPayload.(*scheduler_proto.Job)
	if !ok {
		return errors.Wrap(err, "failed to convert message event proto to *scheduler_proto.Container")
	}
	job := &models.Container{}
	if err := job.FromProto(jobProto); err != nil {
		return errors.Wrap(err, "job.FromProto failed")
	}
	// Verify that the Container has terminated.
	if !job.IsTerminated() {
		return nil
	}

	s.tasksForClientsMux.RLock()
	ch, ok := s.tasksForClients[job.ID]
	s.tasksForClientsMux.RUnlock()
	if ok {
		// Close the corresponding Tasks channel as there can't be any future Tasks for this terminated Container.
		close(ch)
	}
	return nil
}

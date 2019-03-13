package server

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/events"
)

// HandleTaskCreated sends the created Task to the corresponding channels.
func (s *SchedulerServer) HandleTaskCreated(eventProto *events_proto.Event) error {
	eventPayload, err := events.NewModelProtoFromEvent(eventProto)
	if err != nil {
		return errors.Wrap(err, "events.NewModelProtoFromEvent failed")
	}
	taskProto, ok := eventPayload.(*scheduler_proto.Task)
	if !ok {
		return errors.Wrap(err, "failed to convert message event proto to *scheduler_proto.Task")
	}
	s.tasksForNodesMux.RLock()
	ch, ok := s.tasksForNodes[taskProto.NodeId]
	s.tasksForNodesMux.RUnlock()
	if ok {
		ch <- taskProto
	}

	s.tasksForClientsMux.RLock()
	ch, ok = s.tasksForClients[taskProto.JobId]
	s.tasksForClientsMux.RUnlock()
	if ok {
		ch <- taskProto
	}
	return nil
}

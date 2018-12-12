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
	if ch, ok := s.NewTasksByNodeID[taskProto.NodeId]; ok {
		ch <- taskProto
	}
	if ch, ok := s.NewTasksByJobID[taskProto.JobId]; ok {
		ch <- taskProto
	}

	return nil
}

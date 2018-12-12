package server

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
)

// HandleNodeDisconnected updates statuses of the corresponding Tasks.
func (s *SchedulerServer) HandleNodeDisconnected(eventProto *events_proto.Event) error {
	eventPayload, err := events.NewModelProtoFromEvent(eventProto)
	if err != nil {
		return errors.Wrap(err, "events.NewModelProtoFromEvent failed")
	}
	nodeProto, ok := eventPayload.(*scheduler_proto.Node)
	if !ok {
		return errors.New("failed to convert message event proto to *scheduler_proto.Node")
	}

	var tasks models.Tasks
	tx := s.DB.Begin()
	tasksEvents, err := tasks.UpdateStatusForDisconnectedNode(tx, nodeProto.Id)
	if err != nil {
		return errors.Wrap(err, "tasks.UpdateStatusForDisconnectedNode failed")
	}
	if err := events.CommitAndPublish(tx, s.Producer, tasksEvents...); err != nil {
		return errors.Wrap(err, "events.CommitAndPublish failed")
	}
	return nil
}

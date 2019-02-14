package server

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"

	"github.com/mennanov/scalemate/shared/events"
)

// HandleNodeConnected schedules pending Jobs on the connected Node.
func (s *SchedulerServer) HandleNodeConnected(eventProto *events_proto.Event) error {
	eventPayload, err := events.NewModelProtoFromEvent(eventProto)
	if err != nil {
		return errors.Wrap(err, "events.NewModelProtoFromEvent failed")
	}
	nodeProto, ok := eventPayload.(*scheduler_proto.Node)
	if !ok {
		return errors.New("failed to convert message event proto to *scheduler_proto.Node")
	}
	node := &models.Node{}
	if err := node.FromProto(nodeProto); err != nil {
		return errors.Wrap(err, "node.FromProto failed")
	}
	// Populate the node struct fields from DB.
	if err := node.LoadFromDB(s.DB); err != nil {
		return errors.Wrap(err, "node.LoadFromDB failed")
	}

	tx := s.DB.Begin()
	schedulingEvents, err := node.SchedulePendingJobs(tx)
	if err != nil {
		return errors.Wrap(err, "node.SchedulePendingJobs failed")
	}
	if len(schedulingEvents) == 0 {
		s.logger.Debug("no Jobs have been scheduled to the recently connected Node")
	}
	if err := events.CommitAndPublish(tx, s.Producer, schedulingEvents...); err != nil {
		return errors.Wrap(err, "failed to send and commit events")
	}
	return nil
}

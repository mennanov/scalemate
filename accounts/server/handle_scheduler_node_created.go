package server

import (
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

// HandleSchedulerNodeCreatedEvent handles Node created event from Scheduler service.
// It creates a new record in DB for the Node.
func (s AccountsServer) HandleSchedulerNodeCreatedEvent(eventProto *events_proto.Event) error {
	eventNode, ok := eventProto.Payload.(*events_proto.Event_SchedulerNode)
	if !ok {
		return errors.New("failed to cast eventProto.Payload to *events_proto.Event_SchedulerNode")
	}
	node := &models.Node{}
	node.FromSchedulerProto(eventNode.SchedulerNode)

	tx := s.db.Begin()
	event, err := node.Create(tx)
	if err != nil {
		return errors.Wrap(err, "node.Create failed")
	}
	if err := events.CommitAndPublish(tx, s.producer, event); err != nil {
		return errors.Wrap(err, "events.CommitAndPublish failed")
	}
	return nil
}

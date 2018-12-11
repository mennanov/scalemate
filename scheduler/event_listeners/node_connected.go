package event_listeners

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/events"
)

const (
	// NodeConnectedEventsQueueName is an AMQP queue name that is used by NodeConnectedHandler. The named queue is
	// needed to route a message to one consumer at a time.
	NodeConnectedEventsQueueName = "scheduler_node_connected"
)

// NodeConnectedAMQPEventListener schedules pending Jobs on the connected Node.
var NodeConnectedAMQPEventListener = &AMQPEventListener{
	ExchangeName: events.SchedulerAMQPExchangeName,
	QueueName:    NodeConnectedEventsQueueName,
	RoutingKey:   "scheduler.node.updated.#.connected_at.#",
	Handler: func(s *server.SchedulerServer, eventProto *events_proto.Event) error {
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
			logrus.Warn("no schedulingEvents")
		}
		if err := events.CommitAndPublish(tx, s.Publisher, schedulingEvents...); err != nil {
			return errors.Wrap(err, "failed to send and commit events")
		}
		return nil
	},
}

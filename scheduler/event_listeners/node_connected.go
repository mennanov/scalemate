package event_listeners

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// NodeConnectedEventsQueueName is an AMQP queue name that is used by NodeConnectedHandler. The named queue is
	// needed to route a message to one consumer at a time.
	NodeConnectedEventsQueueName = "scheduler_node_connected"
)

// NodeConnectedAMQPEventListener schedules pending Jobs on the connected Node.
var NodeConnectedAMQPEventListener = &AMQPEventListener{
	ExchangeName: utils.SchedulerAMQPExchangeName,
	QueueName:    NodeConnectedEventsQueueName,
	RoutingKey:   "scheduler.node.updated.#.connected_at.#",
	Handler: func(s *server.SchedulerServer, eventProto *events_proto.Event) error {
		eventPayload, err := utils.NewModelProtoFromEvent(eventProto)
		if err != nil {
			return errors.Wrap(err, "utils.NewModelProtoFromEvent failed")
		}
		nodeProto, ok := eventPayload.(*scheduler_proto.Node)
		if !ok {
			return errors.New("failed to convert message event proto to *scheduler_proto.Node")
		}
		node := &models.Node{}
		if err := node.FromProto(nodeProto); err != nil {
			return errors.Wrap(err, "node.FromProto failed")
		}
		tx := s.DB.Begin()
		// Populate the node struct fields from DB.
		if err := node.LoadFromDB(tx); err != nil {
			return errors.Wrap(err, "node.LoadFromDB failed")
		}

		schedulingEvents, err := node.SchedulePendingJobs(tx)
		if err != nil {
			return errors.Wrap(err, "node.SchedulePendingJobs failed")
		}
		if err := utils.SendAndCommit(tx, s.Publisher, schedulingEvents...); err != nil {
			return errors.Wrap(err, "failed to send and commit events")
		}
		return nil
	},
}

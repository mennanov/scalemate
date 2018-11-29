package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/mennanov/scalemate/shared/events/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// NodeCreatedEventsQueueName is AMQP queue name to be used to receive events about the newly created Nodes by
	// Accounts service.
	NodeCreatedEventsQueueName = "accounts_scheduler_node_created"
)

// HandleNodeCreatedEvents handles events that are emitted by the Scheduler service when a new Node is created.
// This handler populates the local `models.Node` table in DB to make it consistent with the Scheduler service.
func (s *AccountsServer) HandleNodeCreatedEvents() error {
	amqpChannel, err := s.AMQPConnection.Channel()
	defer utils.Close(amqpChannel)
	if err != nil {
		return errors.Wrap(err, "failed to create AMQP channel")
	}

	// Declare a "work" queue to be used by multiple consumers (other Accounts service instances) to make sure one event
	// is processed only once.
	queue, err := amqpChannel.QueueDeclare(
		NodeCreatedEventsQueueName,
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return errors.Wrapf(err, "failed to declare AMQP queue %s", NodeCreatedEventsQueueName)
	}

	// Get all Node CREATED events.
	key := "scheduler.node.created"
	if err = amqpChannel.QueueBind(
		queue.Name,
		key,
		utils.SchedulerAMQPExchangeName,
		false,
		nil); err != nil {
		return errors.Wrapf(err, "failed to bind AMQP Queue for key '%s'", key)
	}

	messages, err := amqpChannel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil)

	if err != nil {
		return errors.Wrap(err, "failed to register AMQP Consumer")
	}

	publisher, err := events.NewAMQPPublisher(s.AMQPConnection, utils.AccountsAMQPExchangeName)
	if err != nil {
		return errors.Wrap(err, "failed to create AMQP publisher instance")
	}

	for msg := range messages {
		go func(msg amqp.Delivery) {
			// Always acknowledge the message because it can only fail in a non-retriable way.
			defer msg.Ack(false)

			eventProto := &events_proto.Event{}
			if err := proto.Unmarshal(msg.Body, eventProto); err != nil {
				logrus.Error("failed to unmarshal events_proto.Event")
				return
			}
			if eventNode, ok := eventProto.Payload.(*events_proto.Event_SchedulerNode); ok {
				node := &models.Node{}
				node.FromSchedulerProto(eventNode.SchedulerNode)

				tx := s.DB.Begin()
				event, err := node.Create(tx)
				if err != nil {
					logrus.WithError(err).WithField("eventKey", key).Error("failed to create a Node from event")
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				if err := utils.SendAndCommit(ctx, tx, publisher, event); err != nil {
					logrus.WithError(err).WithField("event", event).Error("failed to SendAndCommit event")
					return
				}
				fmt.Println("SENT EVENTS!!!")
			} else {
				logrus.Error("failed to cast eventProto.Payload to *events_proto.Event_SchedulerNode")
				return
			}
		}(msg)
	}

	return nil
}

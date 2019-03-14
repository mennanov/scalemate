package events

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Consumer defines an interface that is used to consumer events.
type Consumer interface {
	// Consume is a blocking function that consumes events until the context is Done.
	// It should increment the wg.Add(1) when it starts and call wg.Done() when the function exits.
	Consume(ctx context.Context)
	// Close closes all the corresponding consumer resources.
	Close() error
}

// EventHandler is a function that is capable of handling events.
type EventHandler func(eventProto *events_proto.Event) error

// AMQPConsumer implements Consumer for AMQP.
type AMQPConsumer struct {
	exchangeName string
	queueName    string
	routingKey   string
	handler      EventHandler

	channel  *amqp.Channel
	messages <-chan amqp.Delivery
}

// Compile time interface check.
var _ Consumer = new(AMQPConsumer)

// NewAMQPConsumer creates a new instance of AMQPConsumer.
func NewAMQPConsumer(
	connection *amqp.Connection,
	exchangeName string,
	queueName string,
	routingKey string,
	handler EventHandler,
) (*AMQPConsumer, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "connection.Channel failed")
	}

	consumer, err := NewAMQPRawConsumer(channel, exchangeName, queueName, routingKey)
	if err != nil {
		return nil, errors.Wrap(err, "events.NewAMQPRawConsumer failed")
	}

	return &AMQPConsumer{
		exchangeName: exchangeName,
		queueName:    queueName,
		routingKey:   routingKey,
		handler:      handler,
		channel:      channel,
		messages:     consumer,
	}, nil
}

// Consume receives messages from an AMQP queue, unmarshals the protobuf message and calls the handler.
// The function terminates when the context is Done.
func (l *AMQPConsumer) Consume(ctx context.Context) {
	for {
		select {
		case msg, ok := <-l.messages:
			if !ok {
				logrus.Error("messages channel is unexpectedly closed")
				return
			}
			// Process the message in a go routine.
			go func(msg *amqp.Delivery) {
				eventProto := &events_proto.Event{}
				if err := proto.Unmarshal(msg.Body, eventProto); err != nil {
					logrus.Error("failed to unmarshal events_proto.Event")
					return
				}
				if err := l.handler(eventProto); err != nil {
					logrus.WithError(err).WithField("event", eventProto).
						Error("failed to handle AMQP message")
					// Requeue the message only if it is not redelivered, otherwise it may get into a loop.
					// TODO: this makes the failed message be retried only once. Figure out how to retry multiple times.
					if err := msg.Nack(false, !msg.Redelivered); err != nil {
						logrus.WithError(err).Error("msg.Nack failed")
					}
					return
				}
				if err := msg.Ack(false); err != nil {
					logrus.WithError(err).Error("msg.Ack failed")
				}
			}(&msg)
		case <-ctx.Done():
			return
		}
	}
}

// Close closes the corresponding AMQP channel that is used to receive messages.
func (l *AMQPConsumer) Close() error {
	return l.channel.Close()
}

// NewAMQPRawConsumer declares an AMQP queue, binds a consumer to it and returns a channel to receive messages.
func NewAMQPRawConsumer(
	channel *amqp.Channel,
	exchangeName,
	queueName,
	routingKey string,
) (<-chan amqp.Delivery, error) {
	isTmpQueue := queueName == ""
	queue, err := channel.QueueDeclare(
		queueName,
		!isTmpQueue, // durable is false for temp queues.
		isTmpQueue,  // autoDelete is true for temp queues.
		isTmpQueue,  // exclusive is true for temp queues.
		false,
		nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to declare AMQP queue %s", queueName)
	}

	if err = channel.QueueBind(
		queue.Name,
		routingKey,
		exchangeName,
		false,
		nil); err != nil {
		return nil, errors.Wrapf(err, "failed to bind AMQP Queue for key '%s'", routingKey)
	}

	consumer, err := channel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil)

	if err != nil {
		return nil, errors.Wrap(err, "failed to register AMQP Consumer")
	}
	return consumer, nil
}

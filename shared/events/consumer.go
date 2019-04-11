package events

import (
	"github.com/golang/protobuf/proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/nats-io/go-nats-streaming"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Consumer defines an interface to consume events.
type Consumer interface {
	// Consume starts consuming messages and creates a Subscription to control this process.
	Consume(handlers ...EventHandler) (Subscription, error)
}

// Subscription controls an ongoing messages consumption.
type Subscription interface {
	// Close stops receiving messages and closes the corresponding resources if any.
	Close() error
	// Errors returns a channel of errors that occur during the messages processing.
	Errors() <-chan error
}

// EventHandler defines an interface for matching and processing events.
type EventHandler interface {
	// Handle processes the given event.
	Handle(eventProto *events_proto.Event) error
}

// createStanMsgHandler returns a stan.MsgHandler function that processes the messages via the given handlers and sends
// all errors to the handlersErrors channel.
func createStanMsgHandler(handlersErrors chan error, logger *logrus.Logger, handlers ...EventHandler) stan.MsgHandler {
	return func(msg *stan.Msg) {
		logger.Debug("received message: %s", msg.String())

		event := &events_proto.Event{}
		if err := proto.Unmarshal(msg.Data, event); err != nil {
			handlersErrors <- errors.Wrap(err, "proto.Unmarshal failed")
			return
		}

		allHandlersSucceeded := true
		for _, handler := range handlers {
			if err := handler.Handle(event); err != nil {
				allHandlersSucceeded = false
				handlersErrors <- errors.Wrap(err, "handler.Handle failed")
				break
			}
		}
		// Manually acknowledge the message only if all the handlers succeeded to process it.
		// Unacknowledged message will be redelivered after the AckTimeout.
		if allHandlersSucceeded {
			if err := msg.Ack(); err != nil {
				handlersErrors <- errors.Wrap(err, "failed to acknowledge a message")
			}
		}
	}
}

// NatsConsumer implements a Consumer interface for NATS Streaming.
type NatsConsumer struct {
	conn    stan.Conn
	subject string
	logger  *logrus.Logger
	opts    []stan.SubscriptionOption
}

// NewNatsConsumer creates a new instance of NatsConsumer.
// stan.SetManualAckMode() is forcibly appended to the opts as it is required by the message handler behavior to
// provide data consistency: message is acknowledged only if it was successfully processed by all the handlers.
func NewNatsConsumer(
	conn stan.Conn, subject string, logger *logrus.Logger, opts ...stan.SubscriptionOption) *NatsConsumer {
	return &NatsConsumer{conn: conn, subject: subject, logger: logger, opts: append(opts, stan.SetManualAckMode())}
}

// Consume subscribes to the topic and starts receiving messages until the context is done.
// The messages are converted to events_proto.Event and processed by the given handlers.
func (c *NatsConsumer) Consume(handlers ...EventHandler) (Subscription, error) {
	handlersErrors := make(chan error)
	subscription, err := c.conn.Subscribe(
		c.subject, createStanMsgHandler(handlersErrors, c.logger, handlers...), c.opts...)
	if err != nil {
		return nil, errors.Wrap(err, "conn.Subscribe failed")
	}

	natsSubscription := &NatsSubscription{
		subscription: subscription,
		errors:       handlersErrors,
	}

	return natsSubscription, nil
}

// NatsQueueConsumer implements a Consumer interface for NATS Streaming for queues.
type NatsQueueConsumer struct {
	NatsConsumer
	queueName string
}

// NewNatsQueueConsumer creates a new instance of NatsQueueConsumer. Behavior is identical to the NatsConsumer.
func NewNatsQueueConsumer(
	conn stan.Conn, subject string, queueName string, logger *logrus.Logger, opts ...stan.SubscriptionOption) *NatsQueueConsumer {
	return &NatsQueueConsumer{
		NatsConsumer: NatsConsumer{
			conn: conn, subject: subject, logger: logger, opts: append(opts, stan.SetManualAckMode()),
		},
		queueName: queueName,
	}
}

// Consume subscribes to the topic with a given queue group and starts receiving messages until the context is done.
// The messages are converted to events_proto.Event and processed by the given handlers.
func (c *NatsQueueConsumer) Consume(handlers ...EventHandler) (Subscription, error) {
	handlersErrors := make(chan error)
	subscription, err := c.conn.QueueSubscribe(
		c.subject, c.queueName, createStanMsgHandler(handlersErrors, c.logger, handlers...), c.opts...)
	if err != nil {
		return nil, errors.Wrap(err, "conn.QueueSubscribe failed")
	}

	natsSubscription := &NatsSubscription{
		subscription: subscription,
		errors:       handlersErrors,
	}

	return natsSubscription, nil
}

// NatsSubscription implements a Subscription interface for NATS Streaming.
type NatsSubscription struct {
	subscription stan.Subscription
	errors       chan error
}

// Close closes the existing subscription.
func (s *NatsSubscription) Close() error {
	if err := s.subscription.Close(); err != nil {
		return errors.Wrap(err, "subscription.Close failed")
	}
	close(s.errors)
	return nil
}

// Errors returns errors that occurred during the messages processing.
func (s *NatsSubscription) Errors() <-chan error {
	return s.errors
}

// Compile time interface check.
var _ Subscription = new(NatsSubscription)

// Compile time interface check.
var _ Consumer = new(NatsConsumer)

package event_listeners

import (
	"context"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/events"
)

type EventHandler func(service *server.SchedulerServer, eventProto *events_proto.Event) error

type AMQPEventListener struct {
	ExchangeName string
	QueueName    string
	RoutingKey   string
	Handler      EventHandler
}

// RegisteredEventListeners is a list of all event listeners that needed to be run.
var RegisteredEventListeners = []*AMQPEventListener{
	JobCreatedAMQPEventListener,
	JobPendingAMQPEventListener,
	JobTerminatedAMQPEventListener,
	NodeConnectedAMQPEventListener,
	NodeDisconnectedAMQPEventListener,
	TaskCreatedAMQPEventListener,
	TaskTerminatedAMQPEventListener,
}

// PreparedListener represents an AMQPEventListener that is ready to consume messages.
type PreparedListener struct {
	listener *AMQPEventListener
	consumer <-chan amqp.Delivery
}

// SetUpAMQPEventListeners sets up the AMQP event listeners: declares queues, creates bindings, starts consumers.
func SetUpAMQPEventListeners(listeners []*AMQPEventListener, conn *amqp.Connection) ([]*PreparedListener, error) {
	preparedListeners := make([]*PreparedListener, len(listeners))
	for i, listener := range listeners {
		// FIXME: channel is not closed afterwards. Refactor the code for event listeners as in Accounts service.
		channel, err := conn.Channel()
		if err != nil {
			return nil, errors.Wrap(err, "conn.Channel failed")
		}
		consumer, err := events.NewAMQPRawConsumer(channel, listener.ExchangeName, listener.QueueName, listener.RoutingKey)
		if err != nil {
			return nil, errors.Wrapf(err, "events.NewAMQPRawConsumer failed for listener #%d", i)
		}
		preparedListeners[i] = &PreparedListener{listener: listener, consumer: consumer}
	}
	return preparedListeners, nil
}

// RunEventListeners starts running the provided prepared listeners in a background and exits.
// If the context is cancelled then all the listeners safely stop and signal to the waitGroup.
func RunEventListeners(ctx context.Context, waitGroup *sync.WaitGroup, preparedListeners []*PreparedListener, s *server.SchedulerServer) {
	for _, preparedListener := range preparedListeners {
		go runListener(ctx, waitGroup, preparedListener.consumer, preparedListener.listener.Handler, s)
	}
}

// runListener receives the messages, parses them and calls the handler. The function exits when ctx is Done.
func runListener(ctx context.Context, wg *sync.WaitGroup, messages <-chan amqp.Delivery, handler EventHandler, s *server.SchedulerServer) {
	wg.Add(1)
	// Receive messages, parse them and call the handler.
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				logrus.Error("messages channel is unexpectedly closed")
				wg.Done()
				return
			}
			// Process the message in a go routine.
			go func(msg *amqp.Delivery) {
				eventProto := &events_proto.Event{}
				if err := proto.Unmarshal(msg.Body, eventProto); err != nil {
					logrus.Error("failed to unmarshal events_proto.Event")
					return
				}
				if err := handler(s, eventProto); err != nil {
					logrus.WithError(err).WithField("event", eventProto).Error("failed to handle AMQP message")
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
			wg.Done()
			return
		}
	}
}

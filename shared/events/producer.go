package events

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/nats-io/go-nats-streaming"
	"github.com/pkg/errors"
)

var (
	// ErrNoEvents is returned by Producer.Send when events slice is empty.
	ErrNoEvents = errors.New("no events")
)

// Producer represents an event publisher interface.
type Producer interface {
	// Send sends the given events and blocks until all the messages are confirmed to be sent.
	// Events are grouped into EventsMessage proto and sent as a single message to a message broker.
	Send(events ...*events_proto.Event) error
}

// FakeProducer is a publisher that is meant to be used in tests.
type FakeProducer struct {
	SentEvents []*events_proto.Event
}

// Compile time interface check.
var _ Producer = &FakeProducer{}

// Send fakes sending events and logs them instead.
func (f *FakeProducer) Send(events ...*events_proto.Event) error {
	f.SentEvents = append(f.SentEvents, events...)
	return nil
}

// Close imitates closing the producer.
func (f *FakeProducer) Close() error {
	return nil
}

// NewFakeProducer creates a new FakeProducer.
func NewFakeProducer() *FakeProducer {
	return new(FakeProducer)
}

// NatsProducer implements a Producer interface for NATS.
type NatsProducer struct {
	conn    stan.Conn
	subject string
	// retryLimit is the max number of attempts to be used to publish a message.
	retryLimit int
}

// NewNatsProducer creates a new NatsProducer instance.
func NewNatsProducer(conn stan.Conn, subject string, retryLimit int) *NatsProducer {
	return &NatsProducer{conn: conn, subject: subject, retryLimit: retryLimit}
}

// Send publishes the given events to NATS.
func (n *NatsProducer) Send(events ...*events_proto.Event) error {
	if len(events) == 0 {
		return ErrNoEvents
	}
	eventsMessage := &events_proto.Message{
		Events: events,
	}
	message, err := proto.Marshal(eventsMessage)
	if err != nil {
		return errors.Wrap(err, "proto.Marshal failed")
	}

	for i := 0; i < n.retryLimit+1; i++ {
		err = n.conn.Publish(n.subject, message)
		if err == nil {
			break
		}
		if i < n.retryLimit {
			time.Sleep(time.Second)
		}
	}

	return errors.Wrap(err, "failed to publish a message to NATS")
}

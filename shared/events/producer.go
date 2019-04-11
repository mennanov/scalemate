package events

import (
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
}

// NewNatsProducer creates a new NatsProducer instance.
func NewNatsProducer(conn stan.Conn, subject string) *NatsProducer {
	return &NatsProducer{conn: conn, subject: subject}
}

// Send publishes the given events to NATS.
func (n *NatsProducer) Send(events ...*events_proto.Event) error {
	if len(events) == 0 {
		return ErrNoEvents
	}
	messages := make([][]byte, len(events))
	for i, event := range events {
		serializedEvent, err := proto.Marshal(event)
		if err != nil {
			return errors.Wrapf(err, "proto.Marshal failed for event %s", event.String())
		}
		messages[i] = serializedEvent
	}

	for _, message := range messages {
		if err := n.conn.Publish(n.subject, message); err != nil {
			return errors.Wrap(err, "NatsProducer.conn.Publish failed")
		}
	}
	return nil
}

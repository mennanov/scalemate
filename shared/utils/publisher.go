package utils

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"

	"github.com/mennanov/scalemate/shared/events_proto"
)

var (
	// ErrNoEvents is returned by Publisher.SendAsync and Pubisher.Send when events slice is empty.
	ErrNoEvents = errors.New("no events")
)

// Publisher represents an event publisher interface.
type Publisher interface {
	// SendAsync is used to publish multiple events asynchronously.
	SendAsync(events ...*events_proto.Event) error
	// Send similar to SendAsync, but blocks until all the messages are confirmed.
	Send(events ...*events_proto.Event) error
	// Close is used to free up the publisher resources.
	Close() error
}

// FakePublisher is a publisher that is meant to be used in tests.
type FakePublisher struct {
	SentEvents []*events_proto.Event
}

// Compile time interface check.
var _ Publisher = &FakePublisher{}

// SendAsync fakes sending events and logs them instead.
func (f *FakePublisher) SendAsync(events ...*events_proto.Event) error {
	f.SentEvents = append(f.SentEvents, events...)
	return nil
}

// Send fakes sending events with confirmation and logs them instead.
func (f *FakePublisher) Send(events ...*events_proto.Event) error {
	return f.SendAsync(events...)
}

// Close imitates closing the publisher.
func (f *FakePublisher) Close() error {
	return nil
}

// NewFakePublisher creates a new FakePublisher.
func NewFakePublisher() *FakePublisher {
	return new(FakePublisher)
}

// AMQPPublisher is an AMQP Publisher implementation.
// It sends all the events to a single exchange with the routing key generated from the event itself.
// See RoutingKeyFromEvent() below for details.
type AMQPPublisher struct {
	amqpConnection *amqp.Connection
	// This channel is used for non-confirmation sends only.
	amqpChannel *amqp.Channel
	// AMQP exchange name to sendAsync all the messages to.
	exchangeName string
}

// Compile time interface check.
var _ Publisher = &AMQPPublisher{}

// NewAMQPPublisher creates a new AMQPPublisher instance.
func NewAMQPPublisher(amqpConnection *amqp.Connection, exchangeName string) (*AMQPPublisher, error) {
	amqpChannel, err := amqpConnection.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new AMQP channel")
	}

	if err := AMQPExchangeDeclare(amqpChannel, exchangeName); err != nil {
		return nil, errors.Wrap(err, "failed to declare AMQP exchange")
	}

	return &AMQPPublisher{amqpConnection: amqpConnection, amqpChannel: amqpChannel, exchangeName: exchangeName}, nil
}

// sendAsync sends the actual events over the given AMQP channel.
func (p *AMQPPublisher) sendAsync(amqpChannel *amqp.Channel, events ...*events_proto.Event) error {
	n := len(events)
	if n == 0 {
		return ErrNoEvents
	}
	routingKeys := make([]string, n)
	serializedEvents := make([][]byte, n)

	// Prepare the keys and events first.
	for i, event := range events {
		key, err := RoutingKeyFromEvent(event)
		if err != nil {
			return errors.Wrapf(err, "RoutingKeyFromEvent failed for event #%d", i)
		}
		routingKeys[i] = key

		serializedEvent, err := proto.Marshal(event)
		if err != nil {
			return errors.Wrapf(err, "proto.Marshal failed for event #%d", i)
		}
		serializedEvents[i] = serializedEvent
	}

	// Publish the messages to AMQP.
	for i := 0; i < n; i++ {
		if err := amqpChannel.Publish(
			p.exchangeName,
			routingKeys[i],
			false,
			false,
			amqp.Publishing{
				ContentType: "application/octet-stream",
				Body:        serializedEvents[i],
			},
		); err != nil {
			return errors.Wrapf(err, "failed to publish to AMQP for event #%d", i)
		}
	}
	return nil
}

// SendAsync sends AMQP messages with the given events asynchronously.
func (p *AMQPPublisher) SendAsync(events ...*events_proto.Event) error {
	return p.sendAsync(p.amqpChannel, events...)
}

// Send sends AMQP messages with the given events with AMQP confirmations.
// This method should only be used when there is at least one consumer that is capable of consuming these messages,
// otherwise this method will block indefinitely.
// See https://www.rabbitmq.com/confirms.html for details.
// TODO: check if this is actually the case: will it block if there is no consumers?
func (p *AMQPPublisher) Send(events ...*events_proto.Event) error {
	if len(events) == 0 {
		return ErrNoEvents
	}
	// To sendAsync with confirmations a separate channel is needed, since one channel can't be in a dual mode thus
	// p.amqpChannel can't be used as it is not in the confirmation mode.
	amqpChannel, err := p.amqpConnection.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to create a new AMQP channel")
	}
	// Enter a channel confirmation mode.
	if err := amqpChannel.Confirm(false); err != nil {
		return errors.Wrap(err, "failed to set a confirmation mode on a channel")
	}

	confirmations := make(chan amqp.Confirmation, len(events))
	amqpChannel.NotifyPublish(confirmations)

	if err := p.sendAsync(amqpChannel, events...); err != nil {
		return err
	}

	// Wait till all the messages are confirmed to be delivered.
	for {
		c, closed := <-confirmations
		if !c.Ack {
			return errors.Errorf("AMQP message with delivery tag '%d' could not be confirmed", c.DeliveryTag)
		}
		if closed {
			// If the channel is closed all the confirmations are assumed to be received.
			return nil
		}
	}
}

// Close closes the AMQP channel that is used for async sending.
func (p *AMQPPublisher) Close() error {
	return p.amqpConnection.Close()
}

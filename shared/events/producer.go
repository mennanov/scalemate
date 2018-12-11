package events

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/shared/utils"
)

var (
	// ErrNoEvents is returned by Producer.SendAsync and Pubisher.Send when events slice is empty.
	ErrNoEvents = errors.New("no events")
)

const (
	// AMQPPublishRetryWait sets the waiting time between the AMQP publish retries.
	AMQPPublishRetryWait = time.Millisecond * 500
	// AMQPPublishRetryLimit is the maximum AMQP publish retry limit.
	AMQPPublishRetryLimit = 10
)

// Producer represents an event publisher interface.
type Producer interface {
	// SendAsync is used to publish multiple events asynchronously.
	SendAsync(events ...*events_proto.Event) error
	// Send similar to SendAsync, but blocks until all the messages are confirmed.
	Send(events ...*events_proto.Event) error
	// Close is used to free up the publisher resources.
	Close() error
}

// FakeProducer is a publisher that is meant to be used in tests.
type FakeProducer struct {
	SentEvents []*events_proto.Event
}

// Compile time interface check.
var _ Producer = &FakeProducer{}

// SendAsync fakes sending events and logs them instead.
func (f *FakeProducer) SendAsync(events ...*events_proto.Event) error {
	f.SentEvents = append(f.SentEvents, events...)
	return nil
}

// Send fakes sending events with confirmation and logs them instead.
func (f *FakeProducer) Send(events ...*events_proto.Event) error {
	return f.SendAsync(events...)
}

// Close imitates closing the publisher.
func (f *FakeProducer) Close() error {
	return nil
}

// NewFakePublisher creates a new FakeProducer.
func NewFakePublisher() *FakeProducer {
	return new(FakeProducer)
}

// AMQPProducer is an AMQP Producer implementation.
// It sends all the events to a single exchange with the routing key generated from the event itself.
// See RoutingKeyFromEvent() below for details.
type AMQPProducer struct {
	amqpConnection *amqp.Connection
	// This channel is used for non-confirmation sends only.
	amqpChannel *amqp.Channel
	// AMQP exchange name to sendAsync all the messages to.
	exchangeName string
}

// Compile time interface check.
var _ Producer = &AMQPProducer{}

// NewAMQPProducer creates a new AMQPProducer instance.
func NewAMQPProducer(amqpConnection *amqp.Connection, exchangeName string) (*AMQPProducer, error) {
	amqpChannel, err := amqpConnection.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new AMQP channel")
	}

	if err := AMQPExchangeDeclare(amqpChannel, exchangeName); err != nil {
		return nil, errors.Wrap(err, "failed to declare AMQP exchange")
	}

	return &AMQPProducer{amqpConnection: amqpConnection, amqpChannel: amqpChannel, exchangeName: exchangeName}, nil
}

// sendAsync sends the actual events over the given AMQP channel.
func (p *AMQPProducer) sendAsync(amqpChannel *amqp.Channel, events ...*events_proto.Event) error {
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
		if err := p.publishWithRetry(amqpChannel, routingKeys[i], serializedEvents[i], AMQPPublishRetryLimit); err != nil {
			return errors.Wrapf(err, "failed to publish to AMQP for event #%d", i)
		}
	}
	return nil
}

// publishWithRetry publishes AMQP message with increasing retry timeout.
func (p *AMQPProducer) publishWithRetry(ch *amqp.Channel, routingKey string, body []byte, retryCount int) error {
	var err error
	for i := 1; i <= retryCount; i++ {
		err = ch.Publish(
			p.exchangeName,
			routingKey,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/octet-stream",
				Body:        body,
			},
		)
		if err == nil {
			return nil
		}
		logrus.WithError(err).WithField("routingKey", routingKey).Error("failed to publish AMQP message")
		time.Sleep(AMQPPublishRetryWait * time.Duration(i))
	}
	return err
}

// SendAsync sends AMQP messages with the given events asynchronously.
func (p *AMQPProducer) SendAsync(events ...*events_proto.Event) error {
	return p.sendAsync(p.amqpChannel, events...)
}

// Send sends AMQP messages with the given events with AMQP confirmations.
// This method should only be used when there is at least one consumer that is capable of consuming these messages,
// otherwise this method will block indefinitely.
// See https://www.rabbitmq.com/confirms.html for details.
// TODO: check if this is actually the case: will it block if there is no consumers?
func (p *AMQPProducer) Send(events ...*events_proto.Event) error {
	if len(events) == 0 {
		return ErrNoEvents
	}
	// To sendAsync with confirmations a separate channel is needed, since one channel can't be in a dual mode thus
	// p.amqpChannel can't be used as it is not in the confirmation mode.
	amqpChannel, err := p.amqpConnection.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to create a new AMQP channel")
	}
	defer utils.Close(amqpChannel)

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
func (p *AMQPProducer) Close() error {
	return p.amqpChannel.Close()
}

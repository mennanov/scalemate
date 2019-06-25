package events

import (
	"context"
	"database/sql"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/shared/events_proto"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/shared/utils"
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
}

// EventHandler defines an interface for processing events.
type EventHandler interface {
	// Handle processes the given event. It may produce new events as a result of execution.
	// All handlers share the same DB transaction `tx` which is rolled back in case the returned error is not nil.
	// Consumer is responsible for committing the transaction and publishing new events.
	Handle(tx utils.SqlxExtGetter, eventProto *events_proto.Event) ([]*events_proto.Event, error)
}

// NatsConsumer implements a Consumer interface for NATS Streaming.
type NatsConsumer struct {
	conn     stan.Conn
	subject  string
	producer Producer
	db       *sqlx.DB
	logger   *logrus.Logger
	// Limit of retries in case there is an error while processing a message.
	msgRetryLimit int
	// NATS manual acknowledgement retry limit for a single message.
	ackRetryLimit int
	opts          []stan.SubscriptionOption
}

// NewNatsConsumer creates a new instance of NatsConsumer.
// stan.SetManualAckMode() is forcibly appended to the opts as it is required by the message handler behavior to
// provide data consistency: message is acknowledged only if it was successfully processed by all the handlers.
func NewNatsConsumer(conn stan.Conn, subject string, producer Producer, db *sqlx.DB, logger *logrus.Logger,
	msgRetryLimit int, ackRetryLimit int, opts ...stan.SubscriptionOption, ) *NatsConsumer {
	return &NatsConsumer{
		conn:          conn,
		subject:       subject,
		producer:      producer,
		db:            db,
		logger:        logger,
		msgRetryLimit: msgRetryLimit,
		ackRetryLimit: ackRetryLimit,
		opts:          append(opts, stan.SetManualAckMode()),
	}
}

// createStanMsgHandler returns a stan.MsgHandler function that processes the messages via the given handlers and sends
// all errors to the handlersErrors channel.
func (c *NatsConsumer) createStanMsgHandler(handlers ...EventHandler) stan.MsgHandler {
	return func(msg *stan.Msg) {
		c.logger.Debug("received message: %s", msg.String())

		eventsMessage := &events_proto.Message{}
		if err := proto.Unmarshal(msg.Data, eventsMessage); err != nil {
			// This is a non-retryable error. Acknowledge the message since it will never succeed anyway.
			c.logger.Error(errors.Wrap(err, "proto.Unmarshal failed"))
			if err := ackMsg(msg, c.ackRetryLimit, c.logger); err != nil {
				c.logger.Error(err)
			}
			return
		}

		var (
			tx        *sqlx.Tx
			err       error
			newEvents []*events_proto.Event
		)

	TxLoop:
		for retries := 0; retries <= c.msgRetryLimit; retries++ {
			// Reset the slice as it may contain events from the previously failed transaction.
			newEvents = nil
			// LevelSerializable is used to guarantee data consistency across multiple concurrent processes.
			tx, err = c.db.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
			if err != nil {
				// This error is retryable.
				c.logger.Error(errors.Wrap(err, "failed to start a transaction"))
				continue
			}

			for _, event := range eventsMessage.Events {
				for _, handler := range handlers {
					var handlerEvents []*events_proto.Event
					var err error

					handlerEvents, err = handler.Handle(tx, event)
					if err != nil {
						c.logger.Error(utils.RollbackTransaction(tx, errors.Wrap(err, "handler.Handle failed")))
						// Retry the entire transaction.
						continue TxLoop
					}
					newEvents = append(newEvents, handlerEvents...)
				}
			}
			if err := tx.Commit(); err != nil {
				c.logger.Error(utils.RollbackTransaction(tx, errors.Wrap(err, "handler.Handle failed")))
				// Retry the entire transaction.
				continue TxLoop
			}
		}
		// Transaction is successfully committed by now. Manually acknowledge the message.
		// Unacknowledged messages will be redelivered after the AckTimeout.
		if err := ackMsg(msg, c.ackRetryLimit, c.logger); err != nil {
			c.logger.Error(err)
		}

		if err := c.producer.Send(newEvents...); err != nil {
			// Retry logic is supposed to be implemented by the publisher, so if we get here then all retries have
			// failed.
			// TODO: figure out how to recover from this error as it leads to data inconsistency across services.
			c.logger.Error(errors.Wrap(err, "failed to publish events"))
		}
	}
}

func ackMsg(msg *stan.Msg, retryLimit int, logger *logrus.Logger) error {
	var err error
	for i := 0; i <= retryLimit; i++ {
		err = msg.Ack()
		if err == nil {
			break
		}

		if i < retryLimit {
			logger.WithError(err).Info("retrying ack in 1 second")
			time.Sleep(time.Second)
		}
	}
	return errors.Wrap(err, "failed to acknowledge a message")
}

// Consume subscribes to the topic and starts receiving messages until the context is done.
// The messages are converted to events_proto.Event and processed by the given handlers.
func (c *NatsConsumer) Consume(handlers ...EventHandler) (Subscription, error) {
	subscription, err := c.conn.Subscribe(c.subject, c.createStanMsgHandler(handlers...), c.opts...)
	if err != nil {
		return nil, errors.Wrap(err, "conn.Subscribe failed")
	}

	natsSubscription := &NatsSubscription{
		subscription: subscription,
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
	conn stan.Conn,
	subject string,
	queueName string,
	producer Producer,
	db *sqlx.DB,
	logger *logrus.Logger,
	ackRetryLimit int,
	opts ...stan.SubscriptionOption,
) *NatsQueueConsumer {
	return &NatsQueueConsumer{
		NatsConsumer: NatsConsumer{
			conn:          conn,
			subject:       subject,
			producer:      producer,
			db:            db,
			logger:        logger,
			ackRetryLimit: ackRetryLimit,
			opts:          append(opts, stan.SetManualAckMode()),
		},
		queueName: queueName,
	}
}

// Consume subscribes to the topic with a given queue group and starts receiving messages until the context is done.
// The messages are converted to events_proto.Event and processed by the given handlers.
func (c *NatsQueueConsumer) Consume(handlers ...EventHandler) (Subscription, error) {
	subscription, err := c.conn.QueueSubscribe(c.subject, c.queueName, c.createStanMsgHandler(handlers...), c.opts...)
	if err != nil {
		return nil, errors.Wrap(err, "conn.QueueSubscribe failed")
	}

	natsSubscription := &NatsSubscription{
		subscription: subscription,
	}

	return natsSubscription, nil
}

// NatsSubscription implements a Subscription interface for NATS Streaming.
type NatsSubscription struct {
	subscription stan.Subscription
}

// Close closes the existing subscription.
func (s *NatsSubscription) Close() error {
	if err := s.subscription.Close(); err != nil {
		return errors.Wrap(err, "subscription.Close failed")
	}
	return nil
}

// Compile time interface check.
var _ Subscription = new(NatsSubscription)

// Compile time interface check.
var _ Consumer = new(NatsConsumer)

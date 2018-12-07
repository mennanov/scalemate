package utils

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

const (
	// SchedulerAMQPExchangeName is the name of the exchange to be used to send all the events from this service.
	SchedulerAMQPExchangeName = "scheduler_events"
	// AccountsAMQPExchangeName is an AMQP exchange name for all the Accounts service events.
	AccountsAMQPExchangeName = "accounts_events"
)

// AMQPExchangeDeclare declares an AMQP exchange.
func AMQPExchangeDeclare(ch *amqp.Channel, exchangeName string) error {
	return ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

// AMQPExchangeDeclareAccounts declares an AMQP exchange for the Accounts service.
func AMQPExchangeDeclareAccounts(ch *amqp.Channel) error {
	return AMQPExchangeDeclare(ch, AccountsAMQPExchangeName)
}

// AMQPExchangeDeclareAccounts declares an AMQP exchange for the Accounts service.
func AMQPExchangeDeclareScheduler(ch *amqp.Channel) error {
	return AMQPExchangeDeclare(ch, SchedulerAMQPExchangeName)
}

// ConnectAMQPFromEnv creates a connection to the AMQP service (RabbitMQ).
// `addrEnv` is the name of the environment variable which holds the address in the form of
// "amqp://guest:guest@localhost:5672/".
func ConnectAMQPFromEnv(conf AMQPEnvConf) (*amqp.Connection, error) {
	addr := os.Getenv(conf.Addr)

	var i time.Duration
	for i = 1; i <= 8; i *= 2 {
		conn, err := amqp.Dial(addr)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"addr": addr,
			}).Errorf("Could not connect to AMQP: %s", err)

			d := i * time.Second
			logrus.Infof("Retrying to connect to AMQP in %s sec", d)
			time.Sleep(d)
			continue
		}
		return conn, nil
	}

	return nil, errors.New("Could not connect to AMQP")
}

// NewAMQPConsumer declares an AMQP queue, binds a consumer to it and returns a channel to receive messages.
func NewAMQPConsumer(conn *amqp.Connection, exchangeName, queueName, routingKey string) (<-chan amqp.Delivery, error) {
	amqpChannel, err := conn.Channel()
	defer func() {
		if err := amqpChannel.Close(); err != nil {
			logrus.WithError(err).Error("failed to close AMQP channel")
		}
	}()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AMQP channel")
	}

	queue, err := amqpChannel.QueueDeclare(
		queueName,
		true, // FIXME: figure out if it should be false for temp queues.
		false,
		false,
		false,
		nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to declare AMQP queue %s", queueName)
	}

	if err = amqpChannel.QueueBind(
		queue.Name,
		routingKey,
		exchangeName,
		false,
		nil); err != nil {
		return nil, errors.Wrapf(err, "failed to bind AMQP Queue for key '%s'", routingKey)
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
		return nil, errors.Wrap(err, "failed to register AMQP Consumer")
	}
	return messages, nil
}

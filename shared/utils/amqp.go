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
	return ch.ExchangeDeclare(
		AccountsAMQPExchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

// AMQPExchangeDeclareAccounts declares an AMQP exchange for the Accounts service.
func AMQPExchangeDeclareScheduler(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		SchedulerAMQPExchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
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

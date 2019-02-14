package server

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// WithLogger creates an option that sets the logger.
func WithLogger(logger *logrus.Logger) Option {
	return func(s *AccountsServer) error {
		s.logger = logger
		return nil
	}
}

// WithBCryptCost sets the bCryptCost field value from environment variables.
func WithBCryptCost(cost int) Option {
	return func(s *AccountsServer) error {
		s.bCryptCost = cost
		return nil
	}
}

// WithAccessTokenTTL sets the accessTokenTTL field value from environment variables.
func WithAccessTokenTTL(ttl time.Duration) Option {
	return func(s *AccountsServer) error {
		s.accessTokenTTL = ttl
		return nil
	}
}

// WithRefreshTokenTTL creates an option that sets the refreshTokenTTL field value from environment variables.
func WithRefreshTokenTTL(ttl time.Duration) Option {
	return func(s *AccountsServer) error {
		s.refreshTokenTTL = ttl
		return nil
	}
}

// WithJWTSecretKey creates an option that sets the jwtSecretKey field value.
func WithJWTSecretKey(jwtSecretKey []byte) Option {
	return func(s *AccountsServer) error {
		s.jwtSecretKey = jwtSecretKey
		return nil
	}
}

// WithClaimsInjector creates an option that sets the ClaimsInjector to auth.NewJWTClaimsInjector with the
// provided jwtSecretKey.
func WithClaimsInjector(jwtSecretKey []byte) Option {
	return func(s *AccountsServer) error {
		s.claimsInjector = auth.NewJWTClaimsInjector(jwtSecretKey)
		return nil
	}
}

// WithAMQPProducer creates an option that sets the producer field value to AMQPProducer.
func WithAMQPProducer(conn *amqp.Connection) Option {
	return func(s *AccountsServer) error {
		if conn == nil {
			return errors.New("amqp.Connection is nil")
		}
		producer, err := events.NewAMQPProducer(conn, events.AccountsAMQPExchangeName)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPProducer failed")
		}
		s.producer = producer
		return nil
	}
}

// WithAMQPConsumers creates an option that sets the consumers field value to AMQPConsumer(s).
func WithAMQPConsumers(conn *amqp.Connection) Option {
	return func(s *AccountsServer) error {
		channel, err := conn.Channel()
		defer utils.Close(channel)

		if err != nil {
			return errors.Wrap(err, "failed to open a new AMQP channel")
		}
		// Declare all required exchanges.
		if err := events.AMQPExchangeDeclare(channel, events.AccountsAMQPExchangeName); err != nil {
			return errors.Wrapf(err, "failed to declare AMQP exchange %s", events.AccountsAMQPExchangeName)
		}
		if err := events.AMQPExchangeDeclare(channel, events.SchedulerAMQPExchangeName); err != nil {
			return errors.Wrapf(err, "failed to declare AMQP exchange %s", events.SchedulerAMQPExchangeName)
		}

		nodeConnectedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"accounts_scheduler_node_created",
			"scheduler.node.created",
			s.HandleSchedulerNodeCreatedEvent)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for nodeConnectedConsumer")
		}
		s.consumers = []events.Consumer{nodeConnectedConsumer}

		return nil
	}
}

// WithDBConnection creates an option that sets the DB field to an existing DB connection.
func WithDBConnection(db *gorm.DB) Option {
	return func(s *AccountsServer) error {
		s.db = db
		return nil
	}
}

// Option modifies the SchedulerServer.
type Option func(server *AccountsServer) error

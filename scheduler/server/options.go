package server

import (
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
	return func(s *SchedulerServer) error {
		s.logger = logger
		return nil
	}
}

// WithDBConnection creates an option that sets the DB field to an existing DB connection.
func WithDBConnection(db *gorm.DB) Option {
	return func(s *SchedulerServer) error {
		s.db = db
		return nil
	}
}

// WithClaimsInjector creates an option that sets claimsInjector field value.
func WithClaimsInjector(injector auth.ClaimsInjector) Option {
	return func(s *SchedulerServer) error {
		s.claimsInjector = injector
		return nil
	}
}

// WithProducer creates an options that sets the producer field value.
func WithProducer(producer events.Producer) Option {
	return func(s *SchedulerServer) error {
		s.producer = producer
		return nil
	}
}

// WithAMQPConsumers creates an option that sets the consumers field value to AMQPConsumer(s).
func WithAMQPConsumers(conn *amqp.Connection) Option {
	return func(s *SchedulerServer) error {
		channel, err := conn.Channel()
		defer utils.Close(channel)

		if err != nil {
			return errors.Wrap(err, "failed to open a new AMQP channel")
		}
		// Declare all required exchanges.
		if err := events.AMQPExchangeDeclare(channel, events.SchedulerAMQPExchangeName); err != nil {
			return errors.Wrapf(err, "failed to declare AMQP exchange %s", events.SchedulerAMQPExchangeName)
		}

		jobStatusUpdatedToPendingConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_job_pending",
			"scheduler.job.updated.#.status.#",
			s.HandleJobPending)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for jobStatusUpdatedToPendingConsumer")
		}

		jobTerminatedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"",
			"scheduler.job.updated.#.status.#",
			s.HandleJobTerminated)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for jobTerminatedConsumer")
		}

		nodeConnectedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_node_connected",
			"scheduler.node.updated.#.connected_at.#",
			s.HandleNodeConnected)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for nodeConnectedConsumer")
		}

		nodeDisconnectedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_node_disconnected",
			"scheduler.node.updated.#.disconnected_at.#",
			s.HandleNodeDisconnected)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for nodeDisconnectedConsumer")
		}

		taskCreatedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"",
			"scheduler.task.created",
			s.HandleTaskCreated)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for taskCreatedConsumer")
		}

		taskTerminatedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_task_terminated",
			"scheduler.task.updated.#.status.#",
			s.HandleTaskTerminated)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for taskTerminatedConsumer")
		}

		s.consumers = []events.Consumer{
			jobStatusUpdatedToPendingConsumer,
			jobTerminatedConsumer,
			nodeConnectedConsumer,
			nodeDisconnectedConsumer,
			taskCreatedConsumer,
			taskTerminatedConsumer,
		}

		return nil
	}
}

// Option modifies the SchedulerServer.
type Option func(server *SchedulerServer) error

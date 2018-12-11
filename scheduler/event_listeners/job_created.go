package event_listeners

import "github.com/mennanov/scalemate/shared/events"

const (
	// JobCreatedEventsQueueName is the name of the AMQP queue to be used to receive events about newly created Jobs.
	JobCreatedEventsQueueName = "scheduler_job_created"
)

// JobCreatedAMQPEventListener schedules the Job to any available Nodes.
var JobCreatedAMQPEventListener = &AMQPEventListener{
	ExchangeName: events.SchedulerAMQPExchangeName,
	QueueName:    JobCreatedEventsQueueName,
	RoutingKey:   "scheduler.job.created",
	Handler:      pendingJobHandler,
}

package utils

const (
	// SchedulerAMQPExchangeName is the name of the exchange to be used to send all the events from this service.
	SchedulerAMQPExchangeName = "scheduler_events"
	// AccountsAMQPExchangeName is an AMQP exchange name for all the Accounts service events.
	AccountsAMQPExchangeName = "accounts_events"
)

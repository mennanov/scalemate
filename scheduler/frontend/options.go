package frontend

import (
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
)

// WithLogger creates an option that sets the logger.
func WithLogger(logger *logrus.Logger) Option {
	return func(s *SchedulerFrontend) {
		s.logger = logger
	}
}

// WithDBConnection creates an option that sets the DB field to an existing DB connection.
func WithDBConnection(db *sqlx.DB) Option {
	return func(s *SchedulerFrontend) {
		s.db = db
	}
}

// WithClaimsInjector creates an option that sets claimsInjector field value.
func WithClaimsInjector(injector auth.ClaimsInjector) Option {
	return func(s *SchedulerFrontend) {
		s.claimsInjector = injector
	}
}

// WithProducer creates an option that sets the producer field value.
func WithProducer(producer events.Producer) Option {
	return func(s *SchedulerFrontend) {
		s.producer = producer
	}
}

// Option modifies the SchedulerFrontend.
type Option func(server *SchedulerFrontend)

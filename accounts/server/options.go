package server

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
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

// WithClaimsInjector creates an option that sets the claimsInjector value.
func WithClaimsInjector(injector auth.ClaimsInjector) Option {
	return func(s *AccountsServer) error {
		s.claimsInjector = injector
		return nil
	}
}

// WithProducer creates an option that sets the producer field value.
func WithProducer(producer events.Producer) Option {
	return func(s *AccountsServer) error {
		s.producer = producer
		return nil
	}
}

// WithDBConnection creates an option that sets the DB field to an existing DB connection.
func WithDBConnection(db *sqlx.DB) Option {
	return func(s *AccountsServer) error {
		s.db = db
		return nil
	}
}

// Option modifies the SchedulerServer.
type Option func(server *AccountsServer) error

package handlers

import (
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/shared/events"
)

// NewIndependentHandlers returns list of handlers that are supposed to run separately from the gRPC service itself.
func NewIndependentHandlers(db *sqlx.DB, producer events.Producer, logger *logrus.Logger, commitRetryLimit int) []events.EventHandler {
	return []events.EventHandler{
		NewContainerCreated(logger, db, producer, commitRetryLimit),
	}
}

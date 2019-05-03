package models

import (
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/shared/utils"
)

// ProcessedEvent represents an already processed event by event handlers
type ProcessedEvent struct {
	HandlerName string
	UUID        []byte
}

// NewProcessedEvent create a new instance of ProcessedEvent.
func NewProcessedEvent(handlerName string, event *events_proto.Event) *ProcessedEvent {
	return &ProcessedEvent{HandlerName: handlerName, UUID: event.Uuid}
}

// Create inserts a new ProcessedEvent in DB.
func (e *ProcessedEvent) Create(db utils.SqlxExtGetter) error {
	_, err := db.Exec("INSERT INTO processed_events (handler_name, uuid) VALUES ($1, $2)", e.HandlerName, e.UUID)
	return utils.HandleDBError(err)
}

// Exists returns true if the event already exists in DB.
func (e *ProcessedEvent) Exists(db sqlx.Queryer) (bool, error) {
	var count int
	if err := db.QueryRowx("SELECT COUNT(*) FROM processed_events WHERE handler_name = $1 AND uuid = $2",
		e.HandlerName, e.UUID).Scan(&count); err != nil {
		return false, utils.HandleDBError(err)
	}
	return count > 0, nil
}

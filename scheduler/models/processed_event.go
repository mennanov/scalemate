package models

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/utils"
)

// TODO: this model is not used by consumers since idempotency is guaranteed by transactions. Consider deleting it.
// ProcessedEvent represents an already processed event by event handlers
type ProcessedEvent struct {
	EventUUID []byte
	Retries   int
	CreatedAt time.Time
	UpdatedAt *time.Time
}

// NewProcessedEventFromProto creates a new instance of ProcessedEvent from the given event proto.
func NewProcessedEventFromProto(event *events_proto.Event) *ProcessedEvent {
	return &ProcessedEvent{
		EventUUID: event.Uuid,
	}
}

// Create inserts a new ProcessedEvent in DB.
func (e *ProcessedEvent) Create(db utils.SqlxExtGetter) error {
	data := map[string]interface{}{
		"event_uuid": e.EventUUID,
		"retries":    e.Retries,
	}
	if !e.CreatedAt.IsZero() {
		data["created_at"] = e.CreatedAt
	}
	if e.UpdatedAt != nil {
		data["updated_at"] = e.UpdatedAt
	}

	query, args, err := psq.Insert("processed_events").SetMap(data).Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}
	return utils.HandleDBError(db.Get(e, query, args...))
}

// Exists returns true if the event already exists in DB.
func (e *ProcessedEvent) Exists(db sqlx.Queryer) (bool, error) {
	var count int
	query, args, err := psq.Select("COUNT(*)").From("processed_events").Where("event_uuid = ?", e.EventUUID).ToSql()
	if err != nil {
		return false, errors.WithStack(err)
	}
	if err := db.QueryRowx(query, args...).Scan(&count); err != nil {
		return false, utils.HandleDBError(err)
	}
	return count > 0, nil
}

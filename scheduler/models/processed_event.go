package models

import (
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/shared/utils"
)

// ProcessedEvent represents an already processed event by event handlers
type ProcessedEvent struct {
	HandlerName string `gorm:"primary_key"`
	UUID        []byte `gorm:"primary_key"`
}

// NewProcessedEvent create a new instance of ProcessedEvent.
func NewProcessedEvent(handlerName string, event *events_proto.Event) *ProcessedEvent {
	return &ProcessedEvent{HandlerName: handlerName, UUID: event.Uuid}
}

// Create inserts a new ProcessedEvent in DB.
func (e *ProcessedEvent) Create(db *gorm.DB) error {
	return utils.HandleDBError(db.Create(e))
}

// Exists returns true if the event already exists in DB.
func (e *ProcessedEvent) Exists(db *gorm.DB) (bool, error) {
	var count int
	if err := utils.HandleDBError(
		db.Model(e).
			Where("handler_name = ? AND uuid = ?", e.HandlerName, e.UUID).Count(&count)); err != nil {
		return false, err
	}
	return count > 0, nil
}

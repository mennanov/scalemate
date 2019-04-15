package handlers

import (
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// NodeDisconnectedHandler marks the running Tasks on the Node as failed.
type NodeDisconnectedHandler struct {
	handlerName string
	db          *gorm.DB
	producer    events.Producer
}

// NewNodeDisconnectedHandler creates a new instance of NodeDisconnectedHandler.
func NewNodeDisconnectedHandler(
	handlerName string,
	db *gorm.DB,
	producer events.Producer,
) *NodeDisconnectedHandler {
	return &NodeDisconnectedHandler{handlerName: handlerName, db: db, producer: producer}
}

// Handle updates statuses of the corresponding Tasks.
func (s *NodeDisconnectedHandler) Handle(eventProto *events_proto.Event) error {
	if eventProto.Type != events_proto.Event_UPDATED {
		return nil
	}
	nodePayload, ok := eventProto.Payload.(*events_proto.Event_SchedulerNode)
	if !ok {
		return nil
	}
	nodeProto := nodePayload.SchedulerNode
	if eventProto.PayloadMask == nil {
		return nil
	}
	if !in("status", eventProto.PayloadMask.Paths) {
		return nil
	}
	if nodeProto.Status != scheduler_proto.Node_STATUS_OFFLINE {
		// Ignore this event since the Node is not offline.
		return nil
	}
	processedEvent := models.NewProcessedEvent(s.handlerName, eventProto)
	exists, err := processedEvent.Exists(s.db)
	if err != nil {
		return errors.Wrap(err, "processedEvent.Exists failed")
	}
	if exists {
		// Event has been already processed.
		return nil
	}

	var tasks models.Tasks
	tx := s.db.Begin()
	tasksEvents, err := tasks.UpdateStatusForDisconnectedNode(tx, nodeProto.Id)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "tasks.UpdateStatusForDisconnectedNode failed"))
	}
	if err := processedEvent.Create(tx); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "processedEvent.Create failed"))
	}
	if err := events.CommitAndPublish(tx, s.producer, tasksEvents...); err != nil {
		return errors.Wrap(err, "events.CommitAndPublish failed")
	}
	return nil
}

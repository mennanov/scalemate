package handlers

import (
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"

	"github.com/mennanov/scalemate/shared/events"
)

// NodeUpdatedHandler schedules all pending Jobs to be run on the Node if it is online.
type NodeUpdatedHandler struct {
	handlerName string
	db          *gorm.DB
	producer    events.Producer
	logger      *logrus.Logger
}

// NewNodeUpdatedHandler creates a new NodeUpdatedHandler instance.
func NewNodeUpdatedHandler(
	handlerName string,
	db *gorm.DB,
	producer events.Producer,
	logger *logrus.Logger,
) *NodeUpdatedHandler {
	return &NodeUpdatedHandler{handlerName: handlerName, db: db, producer: producer, logger: logger}
}

func in(what string, where []string) bool {
	for _, s := range where {
		if what == s {
			return true
		}
	}
	return false
}

// Handle schedules pending Jobs on the connected Node.
func (s *NodeUpdatedHandler) Handle(eventProto *events_proto.Event) error {
	if eventProto.Type != events_proto.Event_UPDATED {
		return nil
	}
	eventPayload, ok := eventProto.Payload.(*events_proto.Event_SchedulerNode)
	if !ok {
		return nil
	}
	if eventProto.PayloadMask == nil {
		return nil
	}
	if !in("status", eventProto.PayloadMask.Paths) {
		return nil
	}
	nodeProto := eventPayload.SchedulerNode
	if nodeProto.Status != scheduler_proto.Node_STATUS_ONLINE {
		// Ignore this event since the Node went offline.
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
	node := &models.Node{}
	if err := node.FromProto(nodeProto); err != nil {
		return errors.Wrap(err, "node.FromProto failed")
	}
	tx := s.db.Begin()
	// Populate the node struct fields from DB.
	if err := node.LoadFromDBForUpdate(s.db); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "node.LoadFromDBForUpdate failed"))
	}

	schedulingEvents, err := node.SchedulePendingJobs(tx)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "node.SchedulePendingJobs failed"))
	}
	if err := processedEvent.Create(tx); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "processedEvent.Create failed"))
	}
	if err := events.CommitAndPublish(tx, s.producer, schedulingEvents...); err != nil {
		return errors.Wrap(err, "failed to send and commit events")
	}
	return nil
}

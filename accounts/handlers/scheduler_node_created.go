package handlers

import (
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// SchedulerNodeCreatedHandler handles Node created event from the Scheduler service.
type SchedulerNodeCreatedHandler struct {
	db       *gorm.DB
	producer events.Producer
	logger   *logrus.Logger
}

// NewSchedulerNodeCreatedHandler returns a new instance of SchedulerNodeCreatedHandler.
func NewSchedulerNodeCreatedHandler(
	db *gorm.DB, producer events.Producer, logger *logrus.Logger) *SchedulerNodeCreatedHandler {
	return &SchedulerNodeCreatedHandler{db: db, producer: producer, logger: logger}
}

// Handle creates a new Node record in DB to avoid RPC to the Scheduler service in order to authenticate a Node.
func (s *SchedulerNodeCreatedHandler) Handle(eventProto *events_proto.Event) error {
	s.logger.Trace("SchedulerNodeCreatedHandler.Handle()")
	if eventProto.Type != events_proto.Event_CREATED {
		s.logger.WithField("event", eventProto.String()).
			Debug("Skipping event of a type different than Event_CREATED")
		return nil
	}
	eventNode, ok := eventProto.Payload.(*events_proto.Event_SchedulerNode)
	if !ok {
		return nil
	}
	s.logger.WithField("event", eventProto.String()).Info("processing a Node created event")

	node := &models.Node{}
	node.FromSchedulerProto(eventNode.SchedulerNode)

	tx := s.db.Begin()
	event, err := node.Create(tx)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "node.Create failed"))
	}
	if err := events.CommitAndPublish(tx, s.producer, event); err != nil {
		return errors.Wrap(err, "events.CommitAndPublish failed")
	}
	return nil
}

// Compile time interface check.
var _ events.EventHandler = new(SchedulerNodeCreatedHandler)

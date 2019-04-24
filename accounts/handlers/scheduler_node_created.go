package handlers

import (
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// SchedulerNodeCreatedHandler handles Node created event from the Scheduler service.
type SchedulerNodeCreatedHandler struct {
	db       *sqlx.DB
	producer events.Producer
	logger   *logrus.Logger
}

// NewSchedulerNodeCreatedHandler returns a new instance of SchedulerNodeCreatedHandler.
func NewSchedulerNodeCreatedHandler(
	db *sqlx.DB, producer events.Producer, logger *logrus.Logger) *SchedulerNodeCreatedHandler {
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

	node := models.NewNodeFromSchedulerProto(eventNode.SchedulerNode)

	tx, err := s.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	if err := node.Create(tx); err != nil {
		if s, ok := status.FromError(errors.Cause(err)); ok {
			if s.Code() == codes.AlreadyExists {
				// Do nothing if the Node already exists.
				return utils.RollbackTransaction(tx, nil)
			}
		}

		return utils.RollbackTransaction(tx, errors.Wrap(err, "node.Create failed"))
	}
	return tx.Commit()
}

// Compile time interface check.
var _ events.EventHandler = new(SchedulerNodeCreatedHandler)

package handlers

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

// SchedulerNodeCreatedHandler handles Node created event from the Scheduler service.
type SchedulerNodeCreatedHandler struct {
	db *sqlx.DB
}

// NewSchedulerNodeCreatedHandler returns a new instance of SchedulerNodeCreatedHandler.
func NewSchedulerNodeCreatedHandler(db *sqlx.DB) *SchedulerNodeCreatedHandler {
	return &SchedulerNodeCreatedHandler{db: db}
}

// Handle creates a new Node record in DB to avoid RPC to the Scheduler service in order to authenticate a Node.
func (s *SchedulerNodeCreatedHandler) Handle(ctx context.Context, eventProto *events_proto.Event) error {
	eventPayload, ok := eventProto.Payload.(*events_proto.Event_SchedulerNodeCreated)
	if !ok {
		return nil
	}

	node := models.NewNodeFromSchedulerProto(eventPayload.SchedulerNodeCreated.Node)

	if err := node.Create(s.db); err != nil {
		if s, ok := status.FromError(errors.Cause(err)); ok {
			if s.Code() == codes.AlreadyExists {
				// Do nothing if the Node already exists.
				return nil
			}
		}
		return errors.Wrap(err, "node.Create failed")
	}
	return nil
}

// Compile time interface check.
var _ events.EventHandler = new(SchedulerNodeCreatedHandler)

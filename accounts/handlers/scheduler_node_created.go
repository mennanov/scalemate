package handlers

import (
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
	logger *logrus.Logger
}

// NewSchedulerNodeCreatedHandler returns a new instance of SchedulerNodeCreatedHandler.
func NewSchedulerNodeCreatedHandler(logger *logrus.Logger) *SchedulerNodeCreatedHandler {
	return &SchedulerNodeCreatedHandler{logger: logger}
}

// Handle creates a new Node record in DB to avoid RPC to the Scheduler service in order to authenticate a Node.
func (s *SchedulerNodeCreatedHandler) Handle(
	tx utils.SqlxExtGetter,
	eventProto *events_proto.Event,
) ([]*events_proto.Event, error) {
	s.logger.Trace("SchedulerNodeCreatedHandler.Handle()")

	eventPayload, ok := eventProto.Payload.(*events_proto.Event_SchedulerNodeCreated)
	if !ok {
		return nil, nil
	}

	node := models.NewNodeFromSchedulerProto(eventPayload.SchedulerNodeCreated.Node)

	if err := node.Create(tx); err != nil {
		if s, ok := status.FromError(errors.Cause(err)); ok {
			if s.Code() == codes.AlreadyExists {
				// Do nothing if the Node already exists.
				return nil, nil
			}
		}
		return nil, errors.Wrap(err, "node.Create failed")
	}
	return nil, nil
}

// Compile time interface check.
var _ events.EventHandler = new(SchedulerNodeCreatedHandler)

package handlers

import (
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	utils2 "github.com/mennanov/scalemate/accounts/utils"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// JobUpdatedHandler schedules a pending Container on any available node.
type JobUpdatedHandler struct {
	handlerName string
	db          *gorm.DB
	producer    events.Producer
	logger      *logrus.Logger
}

// NewJobUpdatedHandler creates a new JobUpdatedHandler instance.
func NewJobUpdatedHandler(
	handlerName string,
	db *gorm.DB,
	producer events.Producer,
	logger *logrus.Logger,
) *JobUpdatedHandler {
	return &JobUpdatedHandler{handlerName: handlerName, db: db, producer: producer, logger: logger}
}

// Handle schedules the pending Container.
func (h *JobUpdatedHandler) Handle(eventProto *events_proto.Event) error {
	if eventProto.Type != events_proto.Event_UPDATED {
		return nil
	}
	jobProtoPayload, ok := eventProto.Payload.(*events_proto.Event_SchedulerJob)
	if !ok {
		return nil
	}
	jobProto := jobProtoPayload.SchedulerJob
	if jobProto.Status != scheduler_proto.Job_STATUS_PENDING {
		return nil
	}
	processedEvent := models.NewProcessedEvent(h.handlerName, eventProto)
	exists, err := processedEvent.Exists(h.db)
	if err != nil {
		return errors.Wrap(err, "processedEvent.Exists failed")
	}
	if exists {
		// Event has been already processed.
		return nil
	}

	job := &models.Container{}
	if err := job.FromProto(jobProto); err != nil {
		return errors.Wrap(err, "job.FromProto failed")
	}
	tx := h.db.Begin()
	if err := job.LoadFromDBForUpdate(tx); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "job.LoadFromDB failed"))
	}

	node, err := job.NodesAvailable(tx)
	if err != nil {
		wrappedErr := errors.Wrap(err, "job.NodesAvailable failed")
		if st, ok := status.FromError(errors.Cause(err)); ok {
			if st.Code() == codes.NotFound {
				h.logger.WithField("job", job).Info("no suitable Node could be found for a new Container")
				wrappedErr = nil
			}
		}
		return utils.RollbackTransaction(tx, wrappedErr)
	}
	schedulingEvents, err := job.Schedule(tx, node)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "job.Schedule failed"))
	}
	if err := processedEvent.Create(tx); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "processedEvent.Create failed"))
	}
	if err := utils2.CommitAndPublish(tx, h.producer, schedulingEvents...); err != nil {
		return errors.Wrap(err, "events.CommitAndPublish failed")
	}
	return nil
}

// Compile time interface check.
var _ events.EventHandler = new(JobUpdatedHandler)

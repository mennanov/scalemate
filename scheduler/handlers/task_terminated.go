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

// TaskTerminatedHandler updates the corresponding Job status and the corresponding Node's resources.
type TaskTerminatedHandler struct {
	handlerName string
	db          *gorm.DB
	producer    events.Producer
}

// NewTaskTerminatedHandler creates a new instance of TaskTerminatedHandler.
func NewTaskTerminatedHandler(handlerName string, db *gorm.DB, producer events.Producer) *TaskTerminatedHandler {
	return &TaskTerminatedHandler{handlerName: handlerName, db: db, producer: producer}
}

// Handle updates the status of the corresponding Job and the corresponding Node's available resources.
func (s *TaskTerminatedHandler) Handle(eventProto *events_proto.Event) error {
	if eventProto.Type != events_proto.Event_UPDATED {
		return nil
	}
	taskPayload, ok := eventProto.Payload.(*events_proto.Event_SchedulerTask)
	if !ok {
		return nil
	}
	if eventProto.PayloadMask == nil {
		return nil
	}
	if !in("status", eventProto.PayloadMask.Paths) {
		return nil
	}
	taskProto := taskPayload.SchedulerTask
	task := &models.Task{}
	if err := task.FromProto(taskProto); err != nil {
		return errors.Wrap(err, "task.FromProto failed")
	}
	if !task.IsTerminated() {
		// Disregard the Task that is not terminated (no actions are needed in this case).
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

	// Populate the task struct fields from DB.
	if err := task.LoadFromDB(s.db); err != nil {
		return errors.Wrap(err, "task.LoadFromDB failed")
	}

	tx := s.db.Begin()
	// Load the corresponding Job to check if it needs to be rescheduled.
	if err := task.LoadJobFromDBForUpdate(s.db); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "task.LoadJobFromDBForUpdate failed"))
	}

	newJobStatus := scheduler_proto.Job_STATUS_FINISHED
	// Check if the Job needs to be rescheduled on Task failure.
	if task.Job.NeedsRescheduling(scheduler_proto.Task_Status(task.Status)) {
		// Make the Job available for scheduling.
		newJobStatus = scheduler_proto.Job_STATUS_PENDING
	}

	jobStatusUpdatedEvent, err := task.Job.UpdateStatus(tx, newJobStatus)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "failed to update Job status"))
	}

	node := &models.Node{Model: utils.Model{ID: task.NodeID}}
	// Deallocate the Job's resources from the Node.
	nodeUpdates := node.DeallocateJobResources(task.Job)
	if task.Status == utils.Enum(scheduler_proto.Task_STATUS_NODE_FAILED) {
		nodeUpdates["tasks_failed"] = gorm.Expr("tasks_failed + 1")
	} else {
		nodeUpdates["tasks_finished"] = gorm.Expr("tasks_finished + 1")
	}
	nodeUpdatedEvent, err := node.Updates(tx, nodeUpdates)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "failed to update Node statistics"))
	}
	if err := processedEvent.Create(tx); err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "processedEvent.Create failed"))
	}
	if err := events.CommitAndPublish(tx, s.producer, jobStatusUpdatedEvent, nodeUpdatedEvent); err != nil {
		return errors.Wrap(err, "failed to send and commit events")
	}
	return nil
}

package server

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
)

// HandleTaskTerminated updates the status of the corresponding Job and schedules pending Jobs on that Node.
func (s *SchedulerServer) HandleTaskTerminated(eventProto *events_proto.Event) error {
	eventPayload, err := events.NewModelProtoFromEvent(eventProto)
	if err != nil {
		return errors.Wrap(err, "events.NewModelProtoFromEvent failed")
	}
	taskProto, ok := eventPayload.(*scheduler_proto.Task)
	if !ok {
		return errors.Wrap(err, "failed to convert message event proto to *scheduler_proto.Task")
	}
	task := &models.Task{}
	if err := task.FromProto(taskProto); err != nil {
		return errors.Wrap(err, "task.FromProto failed")
	}
	if !task.HasTerminated() {
		// Disregard the Task that is not terminated (no actions are needed in this case).
		return nil
	}
	// Populate the task struct fields from DB.
	if err := task.LoadFromDB(s.db); err != nil {
		return errors.Wrap(err, "task.LoadFromDB failed")
	}

	// Load the corresponding Job to check if it needs to be rescheduled.
	if err := task.LoadJobFromDB(s.db); err != nil {
		return errors.Wrap(err, "task.LoadJobFromDB failed")
	}

	newJobStatus := scheduler_proto.Job_STATUS_FINISHED
	// Check if the Job needs to be rescheduled on Task failure.
	if task.Job.NeedsReschedulingOnTaskFailure(task) {
		// Make the Job available for scheduling.
		newJobStatus = scheduler_proto.Job_STATUS_PENDING
	}

	tx := s.db.Begin()
	jobStatusUpdatedEvent, err := task.Job.UpdateStatus(tx, newJobStatus)
	if err != nil {
		return errors.Wrap(err, "failed to update Job status")
	}

	if err := events.CommitAndPublish(tx, s.producer, jobStatusUpdatedEvent); err != nil {
		return errors.Wrap(err, "failed to send and commit events")
	}
	return nil
}

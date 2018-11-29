package models

import (
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// Task represents a running Job on a Node (Docker container).
type Task struct {
	Model
	Job         *Job
	JobID       uint64 `gorm:"not null;index" sql:"type:integer REFERENCES jobs(id)"`
	Node        *Node
	NodeID      uint64 `gorm:"not null;index" sql:"type:integer REFERENCES nodes(id)"`
	Status      Enum   `gorm:"type:smallint"`
	StartedAt   time.Time
	FinishedAt  time.Time
	ExitCode    int32
	ExitMessage string
}

// Create inserts a new Task in DB and returns a corresponding event.
func (t *Task) Create(db *gorm.DB) (*events_proto.Event, error) {
	if err := utils.HandleDBError(db.Create(t)); err != nil {
		return nil, err
	}
	taskProto, err := t.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "task.ToProto failed")
	}
	event, err := events.NewEventFromPayload(taskProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// ToProto populates the given `*scheduler_proto.Task` with the contents of the Task.
func (t *Task) ToProto(fieldMask *field_mask.FieldMask) (*scheduler_proto.Task, error) {
	p := &scheduler_proto.Task{}
	p.Id = t.ID
	p.JobId = t.JobID
	p.NodeId = t.NodeID
	p.Status = scheduler_proto.Task_Status(t.Status)

	createdAt, err := ptypes.TimestampProto(t.CreatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
	}
	p.CreatedAt = createdAt

	updatedAt, err := ptypes.TimestampProto(t.UpdatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
	}
	p.UpdatedAt = updatedAt

	startedAt, err := ptypes.TimestampProto(t.StartedAt)
	if err != nil {
		return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
	}
	p.StartedAt = startedAt

	finishedAt, err := ptypes.TimestampProto(t.FinishedAt)
	if err != nil {
		return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
	}
	p.FinishedAt = finishedAt

	p.ExitCode = t.ExitCode
	p.ExitMessage = t.ExitMessage

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		pFiltered := &scheduler_proto.Task{Id: uint64(t.ID)}
		if err := fieldmask_utils.StructToStruct(mask, p, pFiltered, generator.CamelCase, stringEye); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return pFiltered, nil
	}

	return p, nil
}

// FromProto populates the Task from the given `scheduler_proto.Task`.
func (t *Task) FromProto(p *scheduler_proto.Task) error {
	t.ID = p.GetId()
	t.JobID = p.GetJobId()
	t.NodeID = p.GetNodeId()
	t.Status = Enum(p.GetStatus())

	if p.CreatedAt != nil {
		createdAt, err := ptypes.Timestamp(p.CreatedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		t.CreatedAt = createdAt
	}

	if p.UpdatedAt != nil {
		updatedAt, err := ptypes.Timestamp(p.UpdatedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		t.UpdatedAt = updatedAt
	}

	if p.StartedAt != nil {
		startedAt, err := ptypes.Timestamp(p.StartedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		t.StartedAt = startedAt
	}

	if p.FinishedAt != nil {
		finishedAt, err := ptypes.Timestamp(p.FinishedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		t.FinishedAt = finishedAt
	}

	t.ExitCode = p.ExitCode
	t.ExitMessage = p.ExitMessage
	return nil
}

// LoadFromDB performs a lookup by ID and populates the struct.
func (t *Task) LoadFromDB(db *gorm.DB, fields ...string) error {
	if t.ID == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't lookup Task with no ID"))
	}
	query := db
	if len(fields) > 0 {
		query = db.Select(fields)
	}
	return utils.HandleDBError(query.First(t, t.ID))
}

// LoadJobFromDB loads the corresponding Job from DB to the Task's Job field.
func (t *Task) LoadJobFromDB(db *gorm.DB, fields ...string) error {
	if t.JobID == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't lookup related Job with no ID"))
	}
	t.Job = &Job{}
	t.Job.ID = t.JobID
	return t.Job.LoadFromDB(db, fields...)
}

// TaskStatusTransitions defines possible Task status transitions.
var TaskStatusTransitions = map[scheduler_proto.Task_Status][]scheduler_proto.Task_Status{
	scheduler_proto.Task_STATUS_UNKNOWN: {
		scheduler_proto.Task_STATUS_RUNNING,
	},
	scheduler_proto.Task_STATUS_RUNNING: {
		scheduler_proto.Task_STATUS_FAILED,
		scheduler_proto.Task_STATUS_FINISHED,
		scheduler_proto.Task_STATUS_NODE_FAILED,
	},
	scheduler_proto.Task_STATUS_FINISHED:    {},
	scheduler_proto.Task_STATUS_FAILED:      {},
	scheduler_proto.Task_STATUS_NODE_FAILED: {},
}

// Updates performs an UPDATE SQL query for the Task fields given in the `updates` argument and returns a corresponding
// event.
func (t *Task) Updates(db *gorm.DB, updates map[string]interface{}) (*events_proto.Event, error) {
	if t.ID == 0 {
		return nil, errors.WithStack(status.Error(codes.FailedPrecondition, "can't update not saved Task"))
	}
	if err := utils.HandleDBError(db.Model(t).Updates(updates)); err != nil {
		return nil, err
	}
	fieldMask := &field_mask.FieldMask{Paths: mapKeys(updates)}
	taskProto, err := t.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "task.ToProto failed")
	}
	event, err := events.NewEventFromPayload(taskProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
		fieldMask)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// UpdateStatus checks if the requested status change is allowed and performs an UPDATE query if so.
func (t *Task) UpdateStatus(db *gorm.DB, newStatus scheduler_proto.Task_Status) (*events_proto.Event, error) {
	taskStatusProto := scheduler_proto.Task_Status(t.Status)
	for _, s := range TaskStatusTransitions[taskStatusProto] {
		if s == newStatus {
			return t.Updates(db, map[string]interface{}{
				"status":     Enum(newStatus),
				"updated_at": time.Now(),
			})
		}
	}
	return nil, status.Errorf(codes.FailedPrecondition, "task with status %s can't be updated to %s",
		taskStatusProto.String(), newStatus.String())
}

type Tasks []Task

func (tasks *Tasks) List(db *gorm.DB, r *scheduler_proto.ListTasksRequest) (uint32, error) {
	query := db.Model(&Task{})
	ordering := r.GetOrdering()

	var orderBySQL string
	switch ordering {
	case scheduler_proto.ListTasksRequest_CREATED_AT_ASC:
		orderBySQL = "tasks.created_at"
	case scheduler_proto.ListTasksRequest_CREATED_AT_DESC:
		orderBySQL = "tasks.created_at DESC"
	case scheduler_proto.ListTasksRequest_UPDATED_AT_ASC:
		orderBySQL = "tasks.updated_at"
	case scheduler_proto.ListTasksRequest_UPDATED_AT_DESC:
		orderBySQL = "tasks.updated_at DESC"
	}
	query = query.Order(orderBySQL)

	// Filter by username.
	query = query.Joins("INNER JOIN jobs ON (tasks.job_id = jobs.id)").Where("jobs.username = ?", r.Username)

	// Filter by Job IDs.
	if len(r.JobId) != 0 {
		query = query.Where("tasks.job_id IN (?)", r.JobId)
	}

	// Filter by status.
	if len(r.Status) != 0 {
		enumStatus := make([]Enum, len(r.Status))
		for i, s := range r.Status {
			enumStatus[i] = Enum(s)
		}
		query = query.Where("tasks.status IN (?)", enumStatus)
	}

	// Perform a COUNT query with no limit and offset applied.
	var count uint32
	if err := utils.HandleDBError(query.Count(&count)); err != nil {
		return 0, err
	}

	// Apply offset.
	query = query.Offset(r.GetOffset())

	// Apply limit.
	var limit uint32
	if r.GetLimit() != 0 {
		limit = r.GetLimit()
	} else {
		limit = 50
	}
	query = query.Limit(limit)

	// Perform a SELECT query.
	if err := utils.HandleDBError(query.Find(&tasks)); err != nil {
		return 0, err
	}
	return count, nil

}

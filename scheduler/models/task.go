package models

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// ListTasksMaxLimit is a maximum allowed limit in the SQL query that lists Tasks.
	ListTasksMaxLimit = 300
)

// Task represents a running Job on a Node (Docker container).
type Task struct {
	utils.Model
	Job        *Job
	JobID      uint64 `gorm:"not null;index" sql:"type:integer REFERENCES jobs(id)"`
	Node       *Node
	NodeID     uint64     `gorm:"not null;index" sql:"type:integer REFERENCES nodes(id)"`
	Status     utils.Enum `gorm:"type:smallint"`
	StartedAt  *time.Time
	FinishedAt *time.Time
}

func (t *Task) String() string {
	taskProto, err := t.ToProto(nil)
	if err != nil {
		return fmt.Sprintf("broken Task: %s", err.Error())
	}
	return taskProto.String()
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
	event, err := events.NewEvent(taskProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
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

	if !t.CreatedAt.IsZero() {
		createdAt, err := ptypes.TimestampProto(t.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.CreatedAt = createdAt
	}

	if t.UpdatedAt != nil {
		updatedAt, err := ptypes.TimestampProto(*t.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.UpdatedAt = updatedAt
	}

	if t.StartedAt != nil {
		startedAt, err := ptypes.TimestampProto(*t.StartedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.StartedAt = startedAt
	}

	if t.FinishedAt != nil {
		finishedAt, err := ptypes.TimestampProto(*t.FinishedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.FinishedAt = finishedAt
	}

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask, generator.CamelCase)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		taskProtoFiltered := &scheduler_proto.Task{Id: uint64(t.ID)}
		if err := fieldmask_utils.StructToStruct(mask, p, taskProtoFiltered); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return taskProtoFiltered, nil
	}

	return p, nil
}

// FromProto populates the Task from the given `scheduler_proto.Task`.
func (t *Task) FromProto(p *scheduler_proto.Task) error {
	t.ID = p.GetId()
	t.JobID = p.GetJobId()
	t.NodeID = p.GetNodeId()
	t.Status = utils.Enum(p.GetStatus())

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
		t.UpdatedAt = &updatedAt
	}

	if p.StartedAt != nil {
		startedAt, err := ptypes.Timestamp(p.StartedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		t.StartedAt = &startedAt
	}

	if p.FinishedAt != nil {
		finishedAt, err := ptypes.Timestamp(p.FinishedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		t.FinishedAt = &finishedAt
	}

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
	event, err := events.NewEvent(taskProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
		fieldMask)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// TaskStatusTransitions defines possible Task status transitions.
var TaskStatusTransitions = map[scheduler_proto.Task_Status][]scheduler_proto.Task_Status{
	scheduler_proto.Task_STATUS_NEW: {
		scheduler_proto.Task_STATUS_RUNNING,
		scheduler_proto.Task_STATUS_CANCELLED,
	},
	scheduler_proto.Task_STATUS_RUNNING: {
		scheduler_proto.Task_STATUS_FAILED,
		scheduler_proto.Task_STATUS_FINISHED,
		scheduler_proto.Task_STATUS_NODE_FAILED,
		scheduler_proto.Task_STATUS_CANCELLED,
	},
	scheduler_proto.Task_STATUS_FINISHED:    {},
	scheduler_proto.Task_STATUS_FAILED:      {},
	scheduler_proto.Task_STATUS_NODE_FAILED: {},
	scheduler_proto.Task_STATUS_CANCELLED:   {},
}

// UpdateStatus updates the Task's status. It returns an update event or an error if the newStatus can not be set.
func (t *Task) UpdateStatus(db *gorm.DB, newStatus scheduler_proto.Task_Status) (*events_proto.Event, error) {
	taskStatusProto := scheduler_proto.Task_Status(t.Status)
	for _, s := range TaskStatusTransitions[taskStatusProto] {
		if s == newStatus {
			now := time.Now()
			return t.Updates(db, map[string]interface{}{
				"status":     utils.Enum(newStatus),
				"updated_at": &now,
			})
		}
	}
	return nil, status.Errorf(codes.FailedPrecondition, "task with status %s can't be updated to %s",
		taskStatusProto.String(), newStatus.String())
}

// IsTerminated returns true if the Task has terminated (not running regardless the reason).
func (t *Task) IsTerminated() bool {
	return t.Status != utils.Enum(scheduler_proto.Task_STATUS_NEW) && t.Status != utils.Enum(scheduler_proto.Task_STATUS_RUNNING)
}

// Tasks represent a collection of Tasks with methods working with a collection of Tasks.
type Tasks []Task

// List gets Tasks from DB filtering by the provided request.
// Returns the total number of Tasks that satisfy the criteria and an error.
func (tasks *Tasks) List(db *gorm.DB, request *scheduler_proto.ListTasksRequest) (uint32, error) {
	query := db.Model(&Task{})
	ordering := request.GetOrdering()

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
	query = query.Joins("INNER JOIN jobs ON (tasks.job_id = jobs.id)").Where("jobs.username = ?", request.Username)

	// Filter by Job IDs.
	if len(request.JobId) != 0 {
		query = query.Where("tasks.job_id IN (?)", request.JobId)
	}

	// Filter by status.
	if len(request.Status) != 0 {
		enumStatus := make([]utils.Enum, len(request.Status))
		for i, s := range request.Status {
			enumStatus[i] = utils.Enum(s)
		}
		query = query.Where("tasks.status IN (?)", enumStatus)
	}

	// Perform a COUNT query with no limit and offset applied.
	var count uint32
	if err := utils.HandleDBError(query.Count(&count)); err != nil {
		return 0, err
	}

	// Apply offset.
	query = query.Offset(request.GetOffset())

	// Apply limit.
	var limit uint32
	if request.GetLimit() <= ListTasksMaxLimit {
		limit = request.GetLimit()
	} else {
		limit = ListTasksMaxLimit
	}
	query = query.Limit(limit)

	// Perform a SELECT query.
	if err := utils.HandleDBError(query.Find(tasks)); err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateStatusForDisconnectedNode bulk updates Tasks that are affected by the disconnected Node.
// It updates the status of the Tasks to NODE_FAILED for the currently running Tasks.
// Populates the receiver with updated Tasks.
func (tasks *Tasks) UpdateStatusForDisconnectedNode(db *gorm.DB, nodeID uint64) ([]*events_proto.Event, error) {
	var updateEvents []*events_proto.Event
	fieldMask := &field_mask.FieldMask{Paths: []string{"status"}}

	now := time.Now()
	rows, err := db.Raw(
		"UPDATE tasks SET status = ?, updated_at = ? WHERE node_id = ? AND status IN(?) RETURNING tasks.*",
		// Set clause.
		utils.Enum(scheduler_proto.Task_STATUS_NODE_FAILED), &now,
		// Where clause.
		nodeID, []utils.Enum{utils.Enum(scheduler_proto.Task_STATUS_NEW), utils.Enum(scheduler_proto.Task_STATUS_RUNNING)}).Rows()

	if err != nil {
		return nil, errors.Wrap(err, "failed to update Tasks status to NODE_FAILED")
	}

	defer rows.Close()

	for rows.Next() {
		var task Task
		if err := db.ScanRows(rows, &task); err != nil {
			return nil, errors.Wrap(err, "db.ScanRows failed")
		}
		taskProto, err := task.ToProto(fieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "task.ToProto failed")
		}
		event, err := events.NewEvent(taskProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
			fieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "NewEvent failed")
		}
		updateEvents = append(updateEvents, event)

		*tasks = append(*tasks, task)
	}

	return updateEvents, nil
}

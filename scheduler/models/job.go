package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// Job defines a Job gorm model.
// Whenever a user runs `scalemate run ...` a new Job is created in DB.
type Job struct {
	Model
	Username    string `gorm:"type:varchar(32);index"`
	Status      Enum   `gorm:"type:smallint;not null"`
	DockerImage string `gorm:"not null"`

	// CpuLimit https://docs.docker.com/config/containers/resource_constraints/#cpu
	CpuLimit float32 `gorm:"type:real;not null"`
	CpuClass Enum    `gorm:"type:smallint"`

	// MemoryLimit is a memory limit in Megabytes.
	MemoryLimit uint32 `gorm:"not null"`

	GpuLimit uint32 `gorm:"type:smallint;not null"`
	GpuClass Enum   `gorm:"type:smallint"`

	// DiskLimit is a disk limit in Megabytes including image size, container writable layer and volumes.
	DiskLimit uint32 `gorm:"not null"`
	DiskClass Enum   `gorm:"type:smallint"`
	// RunConfig is `docker run` specific parameters like ports, volumes, etc.
	RunConfig     postgres.Jsonb
	RestartPolicy Enum `gorm:"type:smallint"`
	// Constraint labels.
	CpuLabels      pq.StringArray `gorm:"type:text[]"`
	GpuLabels      pq.StringArray `gorm:"type:text[]"`
	DiskLabels     pq.StringArray `gorm:"type:text[]"`
	MemoryLabels   pq.StringArray `gorm:"type:text[]"`
	UsernameLabels pq.StringArray `gorm:"type:text[]"`
	NameLabels     pq.StringArray `gorm:"type:text[]"`
	OtherLabels    pq.StringArray `gorm:"type:text[]"`

	Tasks []*Task
}

func (j *Job) whereNodeCpuClass(q *gorm.DB) *gorm.DB {
	if j.CpuClass != 0 {
		return q.Where("(? BETWEEN cpu_class_min AND cpu_class)", j.CpuClass)
	}
	return q
}

func (j *Job) whereNodeCpuAvailable(q *gorm.DB) *gorm.DB {
	return q.Where("cpu_available >= ?", j.CpuLimit)
}

func (j *Job) whereNodeCpuModel(q *gorm.DB) *gorm.DB {
	if len(j.CpuLabels) != 0 {
		return q.Where("cpu_model IN (?)", []string(j.CpuLabels))
	}
	return q
}

func (j *Job) whereNodeGpuClass(q *gorm.DB) *gorm.DB {
	if j.GpuClass != 0 {
		return q.Where("(? BETWEEN gpu_class_min AND gpu_class)", j.GpuClass)
	}
	return q
}

func (j *Job) whereNodeGpuAvailable(q *gorm.DB) *gorm.DB {
	return q.Where("gpu_available >= ?", j.GpuLimit)
}

func (j *Job) whereNodeGpuModel(q *gorm.DB) *gorm.DB {
	if len(j.GpuLabels) != 0 {
		return q.Where("gpu_model IN (?)", []string(j.GpuLabels))
	}
	return q
}

func (j *Job) whereNodeMemoryAvailable(q *gorm.DB) *gorm.DB {
	return q.Where("memory_available >= ?", j.MemoryLimit)
}

func (j *Job) whereNodeMemoryModel(q *gorm.DB) *gorm.DB {
	if len(j.MemoryLabels) != 0 {
		return q.Where("memory_model IN (?)", []string(j.MemoryLabels))
	}
	return q
}

func (j *Job) whereNodeDiskClass(q *gorm.DB) *gorm.DB {
	if j.DiskClass != 0 {
		return q.Where("(? BETWEEN disk_class_min AND disk_class)", j.DiskClass)
	}
	return q
}

func (j *Job) whereNodeDiskAvailable(q *gorm.DB) *gorm.DB {
	return q.Where("disk_available >= ?", j.DiskLimit)
}

func (j *Job) whereNodeDiskModel(q *gorm.DB) *gorm.DB {
	if len(j.DiskLabels) != 0 {
		return q.Where("disk_model IN (?)", []string(j.DiskLabels))
	}
	return q
}

func (j *Job) whereNodeUsername(q *gorm.DB) *gorm.DB {
	if len(j.UsernameLabels) != 0 {
		return q.Where("username IN (?)", []string(j.UsernameLabels))
	}
	return q
}

func (j *Job) whereNodeName(q *gorm.DB) *gorm.DB {
	if len(j.NameLabels) != 0 {
		return q.Where("name IN (?)", []string(j.NameLabels))
	}
	return q
}

func (j *Job) whereNodeLabels(q *gorm.DB) *gorm.DB {
	if len(j.OtherLabels) != 0 {
		return q.Where("labels && ARRAY[?]", []string(j.OtherLabels))
	}
	return q
}

// stringEye is a string identity function used to create field masks.
func stringEye(s string) string {
	return s
}

// FromProto populates the Job from the given `scheduler_proto.Job`.
func (j *Job) FromProto(p *scheduler_proto.Job) error {
	j.ID = p.GetId()
	j.Username = p.GetUsername()
	j.Status = Enum(p.GetStatus())
	j.DockerImage = p.GetDockerImage()
	j.CpuLimit = p.GetCpuLimit()
	j.CpuClass = Enum(p.GetCpuClass())
	j.MemoryLimit = p.GetMemoryLimit()
	j.GpuLimit = p.GetGpuLimit()
	j.GpuClass = Enum(p.GetGpuClass())
	j.DiskLimit = p.GetDiskLimit()
	j.DiskClass = Enum(p.GetDiskClass())

	j.CpuLabels = p.GetCpuLabels()
	j.GpuLabels = p.GetGpuLabels()
	j.DiskLabels = p.GetDiskLabels()
	j.MemoryLabels = p.GetMemoryLabels()
	j.UsernameLabels = p.GetUsernameLabels()
	j.NameLabels = p.GetNameLabels()
	j.OtherLabels = p.GetOtherLabels()

	if p.CreatedAt != nil {
		createdAt, err := ptypes.Timestamp(p.CreatedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		j.CreatedAt = createdAt
	}

	if p.UpdatedAt != nil {
		updatedAt, err := ptypes.Timestamp(p.UpdatedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		j.UpdatedAt = updatedAt
	}

	if p.GetRunConfig() != nil {
		m := &jsonpb.Marshaler{}
		runConfigJSON, err := m.MarshalToString(p.GetRunConfig())
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		j.RunConfig = postgres.Jsonb{RawMessage: json.RawMessage(runConfigJSON)}
	} else {
		j.RunConfig = postgres.Jsonb{RawMessage: json.RawMessage([]byte{})}
	}
	j.RestartPolicy = Enum(p.GetRestartPolicy())

	return nil
}

// ToProto populates the given `*scheduler_proto.Job` with the contents of the Job.
func (j *Job) ToProto(fieldMask *field_mask.FieldMask) (*scheduler_proto.Job, error) {
	p := &scheduler_proto.Job{}
	p.Id = j.ID
	p.Username = j.Username
	p.Status = scheduler_proto.Job_Status(j.Status)
	p.DockerImage = j.DockerImage
	p.CpuLimit = j.CpuLimit
	p.CpuClass = scheduler_proto.CPUClass(j.CpuClass)
	p.MemoryLimit = j.MemoryLimit
	p.GpuLimit = j.GpuLimit
	p.GpuClass = scheduler_proto.GPUClass(j.GpuClass)
	p.DiskLimit = j.DiskLimit
	p.DiskClass = scheduler_proto.DiskClass(j.DiskClass)
	p.RestartPolicy = scheduler_proto.Job_RestartPolicy(j.RestartPolicy)
	p.CpuLabels = j.CpuLabels
	p.GpuLabels = j.GpuLabels
	p.DiskLabels = j.DiskLabels
	p.MemoryLabels = j.MemoryLabels
	p.UsernameLabels = j.UsernameLabels
	p.NameLabels = j.NameLabels
	p.OtherLabels = j.OtherLabels

	runConfigValue, err := j.RunConfig.Value()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runConfig value")
	}
	if runConfigValue != nil {
		r := bytes.NewReader(runConfigValue.([]byte))
		m := &jsonpb.Unmarshaler{}
		runConfig := &scheduler_proto.Job_RunConfig{}
		if err := m.Unmarshal(r, runConfig); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal jsonpb")
		}
		p.RunConfig = runConfig
	} else {
		p.RunConfig = nil
	}

	createdAt, err := ptypes.TimestampProto(j.CreatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
	}

	updatedAt, err := ptypes.TimestampProto(j.UpdatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
	}

	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		// Always include Job ID regardless of the mask.
		pFiltered := &scheduler_proto.Job{Id: j.ID}
		if err := fieldmask_utils.StructToStruct(mask, p, pFiltered, generator.CamelCase, stringEye); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return pFiltered, nil
	}

	return p, nil
}

// Create inserts a new Job in DB and returns the corresponding event.
func (j *Job) Create(db *gorm.DB) (*events_proto.Event, error) {
	if err := utils.HandleDBError(db.Create(j)); err != nil {
		return nil, err
	}
	jobProto, err := j.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "job.ToProto failed")
	}
	event, err := events.NewEventFromPayload(jobProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// LoadFromDB performs a lookup by ID and populates the struct.
func (j *Job) LoadFromDB(db *gorm.DB, fields ...string) error {
	if j.ID == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't lookup Job with no ID"))
	}
	query := db
	if len(fields) > 0 {
		query = db.Select(fields)
	}
	return utils.HandleDBError(query.First(j, j.ID))
}

// MarkJobsAsNodeFailed updates the status field of the Job and returns the corresponding event.
func (j *Job) UpdateStatus(db *gorm.DB, status scheduler_proto.Job_Status) (*events_proto.Event, error) {
	j.Status = Enum(status)
	if err := utils.HandleDBError(db.Model(j).Update("status", Enum(status))); err != nil {
		return nil, errors.Wrap(err, "failed to update Job status")
	}
	fieldMask := &field_mask.FieldMask{Paths: []string{"status"}}
	jobProto, err := j.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "job.ToProto failed")
	}
	event, err := events.NewEventFromPayload(jobProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
		fieldMask)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// SuitableNodeExists checks if there exists at least one Node which is online and is capable of running this Job
// regardless of the resources available: they may become available later.
// This method should be called before `FindSuitableNode` method as `FindSuitableNode` will return the same error
// "failed to find a suitable Node..." regardless why the Node could not be found: Node does not exist or it does not
// have sufficient resources.
func (j *Job) SuitableNodeExists(db *gorm.DB) bool {
	q := db.Model(&Node{}).Where("status = ?", Enum(scheduler_proto.Node_STATUS_ONLINE))
	q = j.whereNodeCpuClass(q)
	q = j.whereNodeDiskClass(q)
	q = j.whereNodeGpuClass(q)

	q = j.whereNodeCpuModel(q)
	q = j.whereNodeGpuModel(q)
	q = j.whereNodeDiskModel(q)
	q = j.whereNodeMemoryModel(q)

	q = j.whereNodeUsername(q)
	q = j.whereNodeName(q)
	q = j.whereNodeLabels(q)

	var count uint
	q.Count(&count)
	return count > 0
}

// FindSuitableNode finds the best available Node to schedule this Job on.
// `SuitableNodeExists` should be called before calling `FindSuitableNode` to check whether a Node with the requested
// hardware exists.
func (j *Job) FindSuitableNode(db *gorm.DB) (*Node, error) {
	// Perform a `SELECT ... FOR UPDATE` query to lock the Node table rows of interest as they are going to be updated
	// later below in this function.
	q := db.Set("gorm:query_option", "FOR UPDATE").
		Where("status = ?", Enum(scheduler_proto.Node_STATUS_ONLINE))
	q = j.whereNodeCpuClass(q)
	q = j.whereNodeCpuAvailable(q)
	q = j.whereNodeDiskClass(q)
	q = j.whereNodeDiskAvailable(q)
	q = j.whereNodeGpuClass(q)
	q = j.whereNodeGpuAvailable(q)
	q = j.whereNodeMemoryAvailable(q)

	q = j.whereNodeCpuModel(q)
	q = j.whereNodeGpuModel(q)
	q = j.whereNodeDiskModel(q)
	q = j.whereNodeMemoryModel(q)

	q = j.whereNodeUsername(q)
	q = j.whereNodeName(q)
	q = j.whereNodeLabels(q)

	// Order most available (least busy) Nodes first.
	orderByAvailability := []string{
		"(cpu_available::real / cpu_capacity::real)",
		"(memory_available::real / memory_capacity::real)", // Cast to real, otherwise it's always 0.
		"(disk_available::real / disk_capacity::real)"}

	if j.GpuLimit != 0 {
		// GPU availability.
		orderByAvailability = append(orderByAvailability, "(gpu_available::real / gpu_capacity::real)")
	}

	// Order Nodes by availability, break a tie with round-robin.
	order := fmt.Sprintf("(%s) DESC, GREATEST(connected_at, scheduled_at) ASC",
		strings.Join(orderByAvailability, " * "))

	node := &Node{}
	if err := utils.HandleDBError(q.Order(order).First(node)); err != nil {
		return nil, errors.Wrap(err, "failed to find a suitable Node for scheduling a job")
	}
	return node, nil
}

// ScheduleForNode allocates resources on the given Node and creates a new Task for it.
func (j *Job) ScheduleForNode(db *gorm.DB, node *Node) (*Task, []*events_proto.Event, error) {
	if j.ID == 0 {
		return nil, nil, errors.WithStack(status.Error(codes.FailedPrecondition, "can't schedule unsaved Job"))
	}

	// A new Task for the Job and the Node.
	task := &Task{
		NodeID: node.ID,
		JobID:  j.ID,
		Status: Enum(scheduler_proto.Task_STATUS_UNKNOWN),
	}
	event, err := task.Create(db)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create a new Task")
	}
	schedulingEvents := []*events_proto.Event{event}

	event, err = node.AllocateJobResources(db, j)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to allocate Node resources")
	}
	schedulingEvents = append(schedulingEvents, event)

	// Update the Job status to SCHEDULED.
	event, err = j.UpdateStatus(db, scheduler_proto.Job_STATUS_SCHEDULED)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to update Job status")
	}
	schedulingEvents = append(schedulingEvents, event)

	return task, schedulingEvents, nil
}

// LoadTasksFromDB loads the corresponding Job from DB to the Task's Job field.
func (j *Job) LoadTasksFromDB(db *gorm.DB, fields ...string) error {
	if j.ID == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "not saved Job can not have Tasks"))
	}
	var tasks Tasks
	if err := utils.HandleDBError(db.Model(&Task{}).Where("job_id = ?", j.ID).Find(&tasks)); err != nil {
		return errors.Wrap(err, "failed to select related Tasks for Job")
	}
	for _, task := range tasks {
		j.Tasks = append(j.Tasks, &task)
	}
	return nil
}

// mapKeys returns a slice of map string keys.
func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

type Jobs []Job

// List selects Jobs from DB by the given filtering request and populates the receiver.
// Returns the total number of Jobs that satisfy the criteria and an error.
func (jobs *Jobs) List(db *gorm.DB, request *scheduler_proto.ListJobsRequest) (uint32, error) {
	query := db.Model(&Job{})
	ordering := request.GetOrdering()

	var orderBySQL string
	switch ordering {
	case scheduler_proto.ListJobsRequest_CREATED_AT_ASC:
		orderBySQL = "created_at"
	case scheduler_proto.ListJobsRequest_CREATED_AT_DESC:
		orderBySQL = "created_at DESC"
	case scheduler_proto.ListJobsRequest_UPDATED_AT_ASC:
		orderBySQL = "updated_at"
	case scheduler_proto.ListJobsRequest_UPDATED_AT_DESC:
		orderBySQL = "updated_at DESC"
	}
	query = query.Order(orderBySQL)

	// Filter by username.
	query = query.Where("username = ?", request.Username)

	// Filter by status.
	if len(request.Status) != 0 {
		enumStatus := make([]Enum, len(request.Status))
		for i, s := range request.Status {
			enumStatus[i] = Enum(s)
		}
		query = query.Where("status IN (?)", enumStatus)
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
	if request.GetLimit() != 0 {
		limit = request.GetLimit()
	} else {
		limit = 50
	}
	query = query.Limit(limit)

	// Perform a SELECT query.
	if err := utils.HandleDBError(query.Find(&jobs)); err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateStatusForNodeFailedTasks performs a bulk update on Jobs status field for the given Job IDs whose corresponding
// Tasks have failed due to a Node failure (abrupt disconnect).
// The status is set to PENDING if the Job needs rescheduling on a Node failure or to FINISHED otherwise.
// Jobs receiver is populated with the updated Jobs.
func (jobs *Jobs) UpdateStatusForNodeFailedTasks(db *gorm.DB, jobIDs []uint64) ([]*events_proto.Event, error) {
	fieldMask := &field_mask.FieldMask{Paths: []string{"status"}}

	// Set status to PENDING for Jobs that require rescheduling, set status to FINISHED for those that don't.
	rows, err := db.Raw(
		`UPDATE jobs SET status = (CASE WHEN restart_policy = ? THEN ?::int ELSE ?::int END), updated_at = ? 
		WHERE id IN(?) RETURNING jobs.*`,
		// Set clause.
		Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE), Enum(scheduler_proto.Job_STATUS_PENDING),
		Enum(scheduler_proto.Job_STATUS_FINISHED), time.Now(),
		// Where clause.
		jobIDs).Rows()

	if err != nil {
		return nil, errors.Wrap(err, "failed to update Jobs status to PENDING")
	}

	defer func() {
		if err := rows.Close(); err != nil {
			logrus.WithError(err).Error("rows.Close failed in jobs.UpdateStatusForNodeFailedTasks")
		}
	}()

	var updateEvents []*events_proto.Event
	for rows.Next() {
		var job Job
		if err := db.ScanRows(rows, &job); err != nil {
			return nil, errors.Wrap(err, "db.ScanRows failed")
		}
		jobProto, err := job.ToProto(fieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "job.ToProto failed")
		}
		event, err := events.NewEventFromPayload(jobProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
			fieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "NewEventFromPayload failed")
		}
		updateEvents = append(updateEvents, event)

		*jobs = append(*jobs, job)
	}

	return updateEvents, nil
}

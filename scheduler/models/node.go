package models

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// JobsScheduledForNodeQueryLimit is the number of Jobs to be selected from DB when Jobs are selected to be
	// scheduled on a Node.
	JobsScheduledForNodeQueryLimit = 100
)

// Node represents a physical machine that runs Tasks (scheduled Jobs).
type Node struct {
	Model
	Username     string `gorm:"not null;unique_index:idx_username_name"`
	Name         string `gorm:"not null;unique_index:idx_username_name"`
	Status       Enum
	CpuCapacity  uint32  `gorm:"type:smallint;not null;"`
	CpuAvailable float32 `gorm:"not null;"`
	CpuClass     Enum    `gorm:"type:smallint;not null;"`
	CpuClassMin  Enum    `gorm:"type:smallint;not null;"`
	CpuModel     string  `gorm:"index:idx_cpu_model"`

	MemoryCapacity  uint32 `gorm:"not null;"`
	MemoryAvailable uint32 `gorm:"not null;"`
	MemoryModel     string `gorm:"index:idx_memory_model"`

	GpuCapacity  uint32 `gorm:"type:smallint;not null;"`
	GpuAvailable uint32 `gorm:"type:smallint;not null;"`
	GpuClass     Enum   `gorm:"type:smallint;not null;"`
	GpuClassMin  Enum   `gorm:"type:smallint;not null;"`
	GpuModel     string `gorm:"index:idx_gpu_model"`

	DiskCapacity  uint32 `gorm:"not null;"`
	DiskAvailable uint32 `gorm:"not null;"`
	DiskClass     Enum   `gorm:"type:smallint;not null;"`
	DiskClassMin  Enum   `gorm:"type:smallint;not null;"`
	DiskModel     string `gorm:"index:idx_disk_model"`

	// User defined labels.
	Labels pq.StringArray `gorm:"type:text[]"`

	ConnectedAt    *time.Time
	DisconnectedAt *time.Time
	ScheduledAt    *time.Time
}

func (n *Node) String() string {
	nodeProto, err := n.ToProto(nil)
	if err != nil {
		return fmt.Sprintf("broken Node: %s", err.Error())
	}
	return nodeProto.String()
}

func (n *Node) whereJobCpuClass(q *gorm.DB) *gorm.DB {
	return q.Where("cpu_class = 0 OR cpu_class BETWEEN ? AND ?", n.CpuClassMin, n.CpuClass)
}

func (n *Node) whereJobCpuLimit(q *gorm.DB) *gorm.DB {
	return q.Where("cpu_limit <= ?", n.CpuAvailable)
}

func (n *Node) whereJobCpuLabels(q *gorm.DB) *gorm.DB {
	return q.Where("cpu_labels IS NULL OR ? = ANY(cpu_labels)", n.CpuModel)
}

func (n *Node) whereJobGpuClass(q *gorm.DB) *gorm.DB {
	return q.Where("gpu_class = 0 OR gpu_class BETWEEN ? AND ?", n.GpuClassMin, n.GpuClass)
}

func (n *Node) whereJobGpuLimit(q *gorm.DB) *gorm.DB {
	return q.Where("gpu_limit <= ?", n.GpuAvailable)
}

func (n *Node) whereJobGpuLabels(q *gorm.DB) *gorm.DB {
	return q.Where("gpu_labels IS NULL OR ? = ANY(gpu_labels)", n.GpuModel)
}

func (n *Node) whereJobDiskClass(q *gorm.DB) *gorm.DB {
	return q.Where("disk_class = 0 OR disk_class BETWEEN ? AND ?", n.DiskClassMin, n.DiskClass)
}

func (n *Node) whereJobDiskLimit(q *gorm.DB) *gorm.DB {
	return q.Where("disk_limit <= ?", n.DiskAvailable)
}

func (n *Node) whereJobDiskLabels(q *gorm.DB) *gorm.DB {
	return q.Where("disk_labels IS NULL OR ? = ANY(disk_labels)", n.DiskModel)
}

func (n *Node) whereJobMemoryLimit(q *gorm.DB) *gorm.DB {
	return q.Where("memory_limit <= ?", n.MemoryAvailable)
}

func (n *Node) whereJobMemoryLabels(q *gorm.DB) *gorm.DB {
	return q.Where("memory_labels IS NULL OR ? = ANY(memory_labels)", n.MemoryModel)
}

func (n *Node) whereJobUsernameLabels(q *gorm.DB) *gorm.DB {
	return q.Where("username_labels IS NULL OR ? = ANY(username_labels)", n.Username)
}

func (n *Node) whereJobNameLabels(q *gorm.DB) *gorm.DB {
	return q.Where("name_labels IS NULL OR ? = ANY(name_labels)", n.Name)
}

func (n *Node) whereJobOtherLabels(q *gorm.DB) *gorm.DB {
	return q.Where("other_labels IS NULL OR ARRAY[?] && other_labels", []string(n.Labels))
}

// ToProto populates the given `*scheduler_proto.Node` with the contents of the Node.
func (n *Node) ToProto(fieldMask *field_mask.FieldMask) (*scheduler_proto.Node, error) {
	p := &scheduler_proto.Node{}
	p.Id = n.ID
	p.Username = n.Username
	p.Name = n.Name
	p.Status = scheduler_proto.Node_Status(n.Status)

	p.CpuCapacity = n.CpuCapacity
	p.CpuAvailable = n.CpuAvailable
	p.CpuClass = scheduler_proto.CPUClass(n.CpuClass)
	p.CpuClassMin = scheduler_proto.CPUClass(n.CpuClassMin)
	p.CpuModel = n.CpuModel

	p.MemoryCapacity = n.MemoryCapacity
	p.MemoryAvailable = n.MemoryAvailable
	p.MemoryModel = n.MemoryModel

	p.GpuCapacity = n.GpuCapacity
	p.GpuAvailable = n.GpuAvailable
	p.GpuClass = scheduler_proto.GPUClass(n.GpuClass)
	p.GpuClassMin = scheduler_proto.GPUClass(n.GpuClassMin)
	p.GpuModel = n.GpuModel

	p.DiskCapacity = n.DiskCapacity
	p.DiskAvailable = n.DiskAvailable
	p.DiskClass = scheduler_proto.DiskClass(n.DiskClass)
	p.DiskClassMin = scheduler_proto.DiskClass(n.DiskClassMin)
	p.DiskModel = n.DiskModel

	if n.ConnectedAt != nil {
		connectedAt, err := ptypes.TimestampProto(*n.ConnectedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.ConnectedAt = connectedAt
	}

	if n.DisconnectedAt != nil {
		disconnectedAt, err := ptypes.TimestampProto(*n.DisconnectedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.DisconnectedAt = disconnectedAt
	}

	if n.ScheduledAt != nil {
		scheduledAt, err := ptypes.TimestampProto(*n.ScheduledAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.ScheduledAt = scheduledAt
	}

	if !n.CreatedAt.IsZero() {
		createdAt, err := ptypes.TimestampProto(n.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.CreatedAt = createdAt
	}

	if n.UpdatedAt != nil {
		updatedAt, err := ptypes.TimestampProto(*n.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.UpdatedAt = updatedAt
	}

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask, generator.CamelCase)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		nodeProtoFiltered := &scheduler_proto.Node{Id: uint64(n.ID)}
		if err := fieldmask_utils.StructToStruct(mask, p, nodeProtoFiltered); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return nodeProtoFiltered, nil
	}

	return p, nil
}

// FromProto populates the Node from the given `scheduler_proto.Node`.
func (n *Node) FromProto(p *scheduler_proto.Node) error {
	n.ID = p.GetId()
	n.Username = p.GetUsername()
	n.Name = p.GetName()
	n.Status = Enum(p.GetStatus())

	n.CpuCapacity = p.GetCpuCapacity()
	n.CpuAvailable = p.GetCpuAvailable()
	n.CpuClass = Enum(p.GetCpuClass())
	n.CpuClassMin = Enum(p.GetCpuClassMin())
	n.CpuModel = p.CpuModel

	n.MemoryCapacity = p.GetMemoryCapacity()
	n.MemoryAvailable = p.GetMemoryAvailable()
	n.MemoryModel = p.GetMemoryModel()

	n.GpuCapacity = p.GetGpuCapacity()
	n.GpuAvailable = p.GetGpuAvailable()
	n.GpuClass = Enum(p.GetGpuClass())
	n.GpuClassMin = Enum(p.GetGpuClassMin())
	n.GpuModel = p.GetGpuModel()

	n.DiskCapacity = p.GetDiskCapacity()
	n.DiskAvailable = p.GetDiskAvailable()
	n.DiskClass = Enum(p.GetDiskClass())
	n.DiskClassMin = Enum(p.GetDiskClassMin())
	n.DiskModel = p.GetDiskModel()

	if p.CreatedAt != nil {
		connectedAt, err := ptypes.Timestamp(p.ConnectedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		n.ConnectedAt = &connectedAt
	}

	if p.DisconnectedAt != nil {
		disconnectedAt, err := ptypes.Timestamp(p.DisconnectedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		n.DisconnectedAt = &disconnectedAt
	}

	if p.ScheduledAt != nil {
		scheduledAt, err := ptypes.Timestamp(p.ScheduledAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		n.ScheduledAt = &scheduledAt
	}

	if p.CreatedAt != nil {
		createdAt, err := ptypes.Timestamp(p.CreatedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		n.CreatedAt = createdAt
	}

	if p.UpdatedAt != nil {
		updatedAt, err := ptypes.Timestamp(p.UpdatedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		n.UpdatedAt = &updatedAt
	}
	return nil
}

// Create inserts a new Node in DB and returns a corresponding event.
func (n *Node) Create(db *gorm.DB) (*events_proto.Event, error) {
	if err := utils.HandleDBError(db.Create(n)); err != nil {
		return nil, err
	}
	nodeProto, err := n.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "node.ToProto failed")
	}
	event, err := events.NewEventFromPayload(nodeProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// Updates performs an UPDATE SQL query for the Node fields given in the `updates` argument and returns a corresponding
// event.
func (n *Node) Updates(db *gorm.DB, updates map[string]interface{}) (*events_proto.Event, error) {
	if n.ID == 0 {
		return nil, errors.WithStack(status.Error(codes.FailedPrecondition, "can't update not saved Node"))
	}
	if err := utils.HandleDBError(db.Model(n).Updates(updates)); err != nil {
		return nil, err
	}
	fieldMask := &field_mask.FieldMask{Paths: mapKeys(updates)}
	nodeProto, err := n.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "node.ToProto failed")
	}
	event, err := events.NewEventFromPayload(nodeProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
		fieldMask)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// Get gets the Node from DB by username and Node name.
func (n *Node) Get(db *gorm.DB, username, nodeName string) error {
	return utils.HandleDBError(db.Where("username = ? AND name = ?", username, nodeName).First(n))
}

// LoadFromDB performs a lookup by ID and populates the struct.
func (n *Node) LoadFromDB(db *gorm.DB, fields ...string) error {
	if n.ID == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't lookup Node with no ID"))
	}
	query := db
	if len(fields) > 0 {
		query = db.Select(fields)
	}
	return utils.HandleDBError(query.First(n, n.ID))
}

// AllocateJobResources is a wrapper around the `Updates` method above to conveniently update the Node's available
// resources that are going to be used by the given Job.
func (n *Node) AllocateJobResources(db *gorm.DB, job *Job) (*events_proto.Event, error) {
	if n.CpuAvailable < job.CpuLimit {
		return nil, errors.
			Errorf("failed to allocate Node CPU: %f requested, %f available", job.CpuLimit, n.CpuAvailable)
	}
	if n.MemoryAvailable < job.MemoryLimit {
		return nil, errors.
			Errorf("failed to allocate Node memory: %d requested, %d available", job.MemoryLimit,
				n.MemoryAvailable)
	}
	if n.DiskAvailable < job.DiskLimit {
		return nil, errors.
			Errorf("failed to allocate Node disk: %d requested, %d available", job.DiskLimit, n.DiskAvailable)
	}
	now := time.Now()
	nodeUpdates := map[string]interface{}{
		"cpu_available":    n.CpuAvailable - job.CpuLimit,
		"memory_available": n.MemoryAvailable - job.MemoryLimit,
		"disk_available":   n.DiskAvailable - job.DiskLimit,
		"scheduled_at":     &now,
	}
	if job.GpuLimit > 0 {
		if n.GpuAvailable < job.GpuLimit {
			return nil, errors.
				Errorf("failed to allocate Node GPU: %d requested, %d available", job.GpuLimit, n.GpuAvailable)
		}
		nodeUpdates["gpu_available"] = n.GpuAvailable - job.GpuLimit
	}
	return n.Updates(db, nodeUpdates)
}

// SchedulePendingJobs finds suitable Jobs that can be scheduled on the given Node and selects the best combination of
// these Jobs so that they all fit into the Node.
// If no pending Jobs are found that can fit into the Node, then returned events are empty (nil slice) and error is nil.
// Each Job is then scheduled to the receiver Node.
// Node resources are updated, Tasks are created, Job statuses are updated to SCHEDULED.
func (n *Node) SchedulePendingJobs(db *gorm.DB) ([]*events_proto.Event, error) {
	var jobs Jobs
	if err := jobs.FindPendingForNode(db, n); err != nil {
		return nil, errors.Wrap(err, "jobs.FindPendingForNode failed")
	}
	if len(jobs) == 0 {
		return nil, nil
	}

	res := AvailableResources{
		CpuAvailable:    n.CpuAvailable,
		MemoryAvailable: n.MemoryAvailable,
		GpuAvailable:    n.GpuAvailable,
		DiskAvailable:   n.DiskAvailable,
	}
	// selectedJobs is a bitarray.BitArray of the selected for scheduling Jobs.
	selectedJobs := SelectJobs(jobs, res)
	allEvents := make([]*events_proto.Event, 0)
	for i, job := range jobs {
		if !selectedJobs.GetBit(uint64(i)) {
			// This Job has not been selected for scheduling: disregard it.
			continue
		}
		schedulerEvents, err := job.CreateTask(db, n)
		if err != nil {
			return nil, errors.Wrapf(err, "job.CreateTask failed for Job: %s", job.String())
		}
		allEvents = append(allEvents, schedulerEvents...)
	}
	return allEvents, nil
}

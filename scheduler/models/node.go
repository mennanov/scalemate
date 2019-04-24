package models

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
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
	// ListNodesMaxLimit is a maximum allowed limit in the SqlxGetter query that lists Nodes.
	ListNodesMaxLimit = 300
)

// Node represents a physical machine that runs Tasks (scheduled Containers).
type Node struct {
	utils.Model
	Username     string `gorm:"not null;unique_index:idx_username_name"`
	Name         string `gorm:"not null;unique_index:idx_username_name"`
	Status       utils.Enum
	CpuCapacity  uint32     `gorm:"type:smallint;not null;"`
	CpuAvailable uint32     `gorm:"not null;"`
	CpuClass     utils.Enum `gorm:"type:smallint;not null;"`

	MemoryCapacity  uint32 `gorm:"not null;"`
	MemoryAvailable uint32 `gorm:"not null;"`

	GpuCapacity  uint32     `gorm:"type:smallint;not null;"`
	GpuAvailable uint32     `gorm:"type:smallint;not null;"`
	GpuClass     utils.Enum `gorm:"type:smallint;not null;"`

	DiskCapacity  uint32     `gorm:"not null;"`
	DiskAvailable uint32     `gorm:"not null;"`
	DiskClass     utils.Enum `gorm:"type:smallint;not null;"`

	NetworkIngressCapacity uint32
	NetworkEgressCapacity  uint32

	ContainersFinished uint64 `gorm:"not null;"`
	ContainersFailed   uint64 `gorm:"not null;"`

	ConnectedAt     *time.Time
	DisconnectedAt  *time.Time
	LastScheduledAt *time.Time

	Ipv4Address []byte

	Labels []*NodeLabel
}

// NodeLabel is a Node label.
type NodeLabel struct {
	NodeID uint64 `gorm:"primary_key" sql:"type:integer REFERENCES nodes(id)"`
	Label  string `gorm:"primary_key"`
}

// Create inserts a new NodeLabel in DB.
func (l *NodeLabel) Create(db *gorm.DB) error {
	return utils.HandleDBError(db.Create(l))
}

func (n *Node) String() string {
	nodeProto, err := n.ToProto(nil)
	if err != nil {
		return fmt.Sprintf("broken Node: %s", err.Error())
	}
	return nodeProto.String()
}

// ToProto populates the given `*scheduler_proto.Node` with the contents of the Node.
func (n *Node) ToProto(fieldMask *field_mask.FieldMask) (*scheduler_proto.Node, error) {
	p := &scheduler_proto.Node{
		Id:                     n.ID,
		Username:               n.Username,
		Name:                   n.Name,
		Status:                 scheduler_proto.Node_Status(n.Status),
		CpuCapacity:            n.CpuCapacity,
		CpuAvailable:           n.CpuAvailable,
		CpuClass:               scheduler_proto.CPUClass(n.CpuClass),
		MemoryCapacity:         n.MemoryCapacity,
		MemoryAvailable:        n.MemoryAvailable,
		GpuCapacity:            n.GpuCapacity,
		GpuAvailable:           n.GpuAvailable,
		GpuClass:               scheduler_proto.GPUClass(n.GpuClass),
		DiskCapacity:           n.DiskCapacity,
		DiskAvailable:          n.DiskAvailable,
		DiskClass:              scheduler_proto.DiskClass(n.DiskClass),
		NetworkIngressCapacity: n.NetworkIngressCapacity,
		NetworkEgressCapacity:  n.NetworkEgressCapacity,
		Labels:                 make([]string, len(n.Labels)),
		ContainersFinished:     n.ContainersFinished,
		ContainersFailed:       n.ContainersFailed,
		Ipv4Address:            n.Ipv4Address,
	}

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

	if n.LastScheduledAt != nil {
		scheduledAt, err := ptypes.TimestampProto(*n.LastScheduledAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.LastScheduledAt = scheduledAt
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

// NewNodeFromProto create a new Node instance from a proto message.
func NewNodeFromProto(p *scheduler_proto.Node) (*Node, error) {
	node := &Node{
		Model: utils.Model{
			ID: p.Id,
		},
		Username:               p.Username,
		Name:                   p.Name,
		Status:                 utils.Enum(p.Status),
		CpuCapacity:            p.CpuCapacity,
		CpuAvailable:           p.CpuAvailable,
		CpuClass:               utils.Enum(p.CpuClass),
		MemoryCapacity:         p.MemoryCapacity,
		MemoryAvailable:        p.MemoryAvailable,
		GpuCapacity:            p.GpuCapacity,
		GpuAvailable:           p.GpuAvailable,
		GpuClass:               utils.Enum(p.GpuClass),
		DiskCapacity:           p.DiskCapacity,
		DiskAvailable:          p.DiskAvailable,
		DiskClass:              utils.Enum(p.DiskClass),
		NetworkIngressCapacity: p.NetworkIngressCapacity,
		NetworkEgressCapacity:  p.NetworkEgressCapacity,
		ContainersFinished:     p.ContainersFinished,
		ContainersFailed:       p.ContainersFailed,
		Ipv4Address:            p.Ipv4Address,
		Labels:                 make([]*NodeLabel, len(p.Labels)),
	}

	if p.ConnectedAt != nil {
		connectedAt, err := ptypes.Timestamp(p.ConnectedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		node.ConnectedAt = &connectedAt
	}

	if p.DisconnectedAt != nil {
		disconnectedAt, err := ptypes.Timestamp(p.DisconnectedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		node.DisconnectedAt = &disconnectedAt
	}

	if p.LastScheduledAt != nil {
		scheduledAt, err := ptypes.Timestamp(p.LastScheduledAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		node.LastScheduledAt = &scheduledAt
	}

	if p.CreatedAt != nil {
		createdAt, err := ptypes.Timestamp(p.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		node.CreatedAt = createdAt
	}

	if p.UpdatedAt != nil {
		updatedAt, err := ptypes.Timestamp(p.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		node.UpdatedAt = &updatedAt
	}

	for _, label := range p.Labels {
		node.Labels = append(node.Labels, &NodeLabel{NodeID: node.ID, Label: label})
	}

	return node, nil
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
	event, err := events.NewEvent(nodeProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// Update performs an UPDATE SqlxGetter query for the Node fields given in the `updates` argument and returns a corresponding
// event.
func (n *Node) Update(db *gorm.DB, updates map[string]interface{}) (*events_proto.Event, error) {
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
	event, err := events.NewEvent(nodeProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
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
		return errors.WithStack(status.Error(codes.InvalidArgument, "can't lookup Node with no ID"))
	}
	query := db
	if len(fields) > 0 {
		query = db.Select(fields)
	}
	return utils.HandleDBError(query.First(n, n.ID))
}

// LoadFromDBForUpdate is similar to LoadFromDB(), but locks the Node's row FOR UPDATE.
func (n *Node) LoadFromDBForUpdate(db *gorm.DB, fields ...string) error {
	return n.LoadFromDB(db.Set("gorm:query_option", "FOR UPDATE"), fields...)
}

// AllocateContainerLimit allocates resources requested by the newLimit on the Node.
// If the oldLimit provided, then the current Node's resources are adjusted accordingly.
// This function returns a map that can be passed to the Node.Update() method.
func (n *Node) AllocateContainerLimit(newLimit, oldLimit *Limit) (map[string]interface{}, error) {
	if newLimit == nil {
		return nil, errors.New("new limit can not be nil")
	}
	cpuAvailable := n.CpuAvailable
	memoryAvailable := n.MemoryAvailable
	diskAvailable := n.DiskAvailable
	gpuAvailable := n.GpuAvailable

	if oldLimit != nil {
		cpuAvailable += oldLimit.Cpu
		memoryAvailable += oldLimit.Memory
		diskAvailable += oldLimit.Disk
		gpuAvailable += oldLimit.Gpu
	}

	if cpuAvailable < newLimit.Cpu {
		return nil, errors.
			Errorf("failed to allocate Node CPU: %f requested, %f available", newLimit.Cpu, cpuAvailable)
	}
	if memoryAvailable < newLimit.Memory {
		return nil, errors.
			Errorf("failed to allocate Node memory: %d requested, %d available", newLimit.Memory, memoryAvailable)
	}
	if diskAvailable < newLimit.Disk {
		return nil, errors.
			Errorf("failed to allocate Node disk: %d requested, %d available", newLimit.Disk, diskAvailable)
	}
	if gpuAvailable < newLimit.Gpu {
		return nil, errors.
			Errorf("failed to allocate Node GPU: %d requested, %d available", newLimit.Gpu, gpuAvailable)
	}
	nodeUpdates := map[string]interface{}{
		"cpu_available":    cpuAvailable - newLimit.Cpu,
		"memory_available": memoryAvailable - newLimit.Memory,
		"disk_available":   diskAvailable - newLimit.Disk,
		"gpu_available":    gpuAvailable - newLimit.Gpu,
	}
	return nodeUpdates, nil
}

// DeallocateContainerLimit deallocates resources requested by the limit on the Node.
// This function returns a map that can be passed to the Node.Update() method.
func (n *Node) DeallocateContainerLimit(limit *Limit) map[string]interface{} {
	nodeUpdates := map[string]interface{}{
		"cpu_available":    n.CpuAvailable + limit.Cpu,
		"memory_available": n.MemoryAvailable + limit.Memory,
		"disk_available":   n.DiskAvailable + limit.Disk,
		"gpu_available":    n.GpuAvailable + limit.Gpu,
	}

	return nodeUpdates
}

// SchedulePendingJobs finds suitable Containers that can be scheduled on the given Node and selects the best combination of
// these Containers so that they all fit into the Node.
// If no pending Containers are found that can fit into the Node, then returned events are empty (nil slice) and error is nil.
// Each Container is then scheduled to the receiver Node.
// Node resources are updated, Tasks are created, Container statuses are updated to SCHEDULED.
//func (n *Node) SchedulePendingJobs(gormDB *gorm.DB) ([]*events_proto.Event, error) {
//	var jobs Containers
//	if err := jobs.FindPendingForNode(gormDB, n); err != nil {
//		return nil, errors.Wrap(err, "jobs.FindPendingForNode failed")
//	}
//	if len(jobs) == 0 {
//		return nil, nil
//	}
//
//	res := AvailableResources{
//		Cpu:    n.CpuAvailable,
//		Memory: n.MemoryAvailable,
//		Gpu:    n.GpuAvailable,
//		Disk:   n.DiskAvailable,
//	}
//	// selectedJobs is a bitarray.BitArray of the selected for scheduling Containers.
//	selectedJobs := SelectJobs(jobs, res)
//	var taskCreatedEvents []*events_proto.Event
//	for i, job := range jobs {
//		if !selectedJobs.GetBit(uint64(i)) {
//			// This Container has not been selected for scheduling: disregard it.
//			continue
//		}
//		taskCreatedEvent, err := job.Schedule(gormDB, n)
//		if err != nil {
//			return nil, errors.Wrapf(err, "job.Schedule failed for Container: %s", job.String())
//		}
//		taskCreatedEvents = append(taskCreatedEvents, taskCreatedEvent...)
//	}
//	return taskCreatedEvents, nil
//}

// PendingContainers returns Containers in the PENDING status that can be scheduled on this Node.
func (n *Node) PendingContainers(db *gorm.DB) ([]Container, error) {
	q := db.Set("gorm:query_option", "FOR UPDATE").
		Where("status = ?", utils.Enum(scheduler_proto.Container_PENDING)).
		Where("? BETWEEN cpu_class_min AND cpu_class_max", n.CpuClass).
		Where("? BETWEEN gpu_class_min AND gpu_class_max", n.GpuClass).
		Where("? BETWEEN disk_class_min AND disk_class_max", n.DiskClass).
		Where("ARRAY[?] @> labels", n.Labels).
		Joins("INNER JOIN (SELECT container_id, MAX(id) from container_limits WHERE status = ? AND cpu > 0 group by container_id) as t1 on (t1.container_id = containers.id)", utils.Enum(scheduler_proto.Limit_CONFIRMED))

	q = q.Limit(ContainersScheduledForNodeQueryLimit)

	var containers []Container
	if err := utils.HandleDBError(q.Find(containers)); err != nil {
		return nil, errors.Wrap(err, "failed to FindPendingForNode for the Node")
	}
	return containers, nil
}

// Nodes represents a collection of Nodes.
type Nodes []Node

// List selects Containers from DB by the given filtering request and populates the receiver.
// Returns the total number of Containers that satisfy the criteria and an error.
func (nodes *Nodes) List(db *gorm.DB, request *scheduler_proto.ListNodesRequest) (uint32, error) {
	query := db.Model(&Node{})
	ordering := request.GetOrdering()

	var orderBySQL string
	switch ordering {
	case scheduler_proto.ListNodesRequest_CONNECTED_AT_DESC:
		orderBySQL = "connected_at DESC"
	case scheduler_proto.ListNodesRequest_CONNECTED_AT_ASC:
		orderBySQL = "created_at"
	case scheduler_proto.ListNodesRequest_DISCONNECTED_AT_DESC:
		orderBySQL = "disconnected_at DESC"
	case scheduler_proto.ListNodesRequest_DISCONNECTED_AT_ASC:
		orderBySQL = "disconnected_at"
	case scheduler_proto.ListNodesRequest_SCHEDULED_AT_DESC:
		orderBySQL = "scheduled_at DESC"
	case scheduler_proto.ListNodesRequest_SCHEDULED_AT_ASC:
		orderBySQL = "scheduled_at"
	}
	query = query.Order(orderBySQL)

	// Filter by status.
	if len(request.Status) > 0 {
		enumStatus := make([]utils.Enum, len(request.Status))
		for i, s := range request.Status {
			enumStatus[i] = utils.Enum(s)
		}
		query = query.Where("status IN (?)", enumStatus)
	}

	if request.CpuAvailable > 0 {
		query = query.Where("cpu_available >= ?", request.CpuAvailable)
	}
	if request.CpuClass != scheduler_proto.CPUClass_CPU_CLASS_UNKNOWN {
		enum := utils.Enum(request.CpuClass)
		query = query.Where("cpu_class_min <= ? AND cpu_class >= ?", enum, enum)
	}

	if request.MemoryAvailable > 0 {
		query = query.Where("memory_available >= ?", request.MemoryAvailable)
	}

	if request.GpuAvailable > 0 {
		query = query.Where("gpu_available >= ?", request.GpuAvailable)
	}
	if request.GpuClass != scheduler_proto.GPUClass_GPU_CLASS_UNKNOWN {
		enum := utils.Enum(request.GpuClass)
		query = query.Where("gpu_class_min <= ? AND gpu_class >= ?", enum, enum)
	}

	if request.DiskAvailable > 0 {
		query = query.Where("disk_available >= ?", request.DiskAvailable)
	}
	if request.DiskClass != scheduler_proto.DiskClass_DISK_CLASS_UNKNOWN {
		enum := utils.Enum(request.DiskClass)
		query = query.Where("disk_class_min <= ? AND disk_class >= ?", enum, enum)
	}

	// TODO: add labels constraints.

	if request.ContainersFinished > 0 {
		query = query.Where("container_finished >= ?", request.ContainersFinished)
	}

	if request.ContainersFailed > 0 {
		query = query.Where("container_failed <= ?", request.ContainersFailed)
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
	if request.Limit <= ListNodesMaxLimit {
		limit = request.Limit
	} else {
		limit = ListNodesMaxLimit
	}
	query = query.Limit(limit)

	// Perform a SELECT query.
	if err := utils.HandleDBError(query.Find(nodes)); err != nil {
		return 0, err
	}
	return count, nil
}

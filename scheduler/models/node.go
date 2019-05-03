package models

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	scheduler_proto.Node
}

// NewNodeFromProto create a new Node instance from a proto message.
func NewNodeFromProto(p *scheduler_proto.Node) *Node {
	return &Node{Node: *p}
}

// NodeLabel is a Node label.
type NodeLabel struct {
	NodeID int64
	Label  string
}

// Create inserts a new ContainerLabel in DB.
func (l *NodeLabel) Create(db sqlx.Ext) error {
	_, err := psq.Insert("node_labels").Columns("node_id", "label").Values(l.NodeID, l.Label).RunWith(db).Exec()
	return utils.HandleDBError(err)
}

// ToProto returns a proto Node instance with applied proto field mask (if provided).
func (n *Node) ToProto(fieldMask *types.FieldMask) (*scheduler_proto.Node, error) {
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		return &n.Node, nil
	}

	mask, err := fieldmask_utils.MaskFromPaths(fieldMask.Paths, generator.CamelCase)
	if err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
	}
	protoFiltered := &scheduler_proto.Node{Id: n.Id}
	if err := fieldmask_utils.StructToStruct(mask, &n.Node, protoFiltered); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
	}
	return protoFiltered, nil
}

// Create inserts a new Node in DB and returns the corresponding event.
// It only inserts those fields that are required for a new Node, others are set to default values.
func (n *Node) Create(db utils.SqlxExtGetter) (*events_proto.Event, error) {
	// NOT NULL columns are always used.
	columns := []string{
		"username",
		"name",
		"status",
		"cpu_capacity", "cpu_available", "cpu_class",
		"memory_capacity", "memory_available",
		"gpu_capacity", "gpu_available", "gpu_class",
		"disk_capacity", "disk_available", "disk_class",
		"network_ingress_capacity", "network_egress_capacity",
		"containers_finished", "containers_failed",
		"fingerprint",
	}
	values := []interface{}{
		n.Username,
		n.Name,
		int32(n.Status),
		n.CpuCapacity, n.CpuAvailable, n.CpuClass,
		n.MemoryCapacity, n.MemoryAvailable,
		n.GpuCapacity, n.GpuAvailable, n.GpuClass,
		n.DiskCapacity, n.DiskAvailable, n.DiskClass,
		n.NetworkIngressCapacity, n.NetworkEgressCapacity,
		n.ContainersFinished, n.ContainersFailed,
		n.Fingerprint,
	}

	// Nullable columns are used only when they have a value.
	if n.ConnectedAt != nil {
		columns = append(columns, "connected_at")
		values = append(values, *n.ConnectedAt)
	}
	if n.DisconnectedAt != nil {
		columns = append(columns, "disconnected_at")
		values = append(values, *n.DisconnectedAt)
	}
	if n.LastScheduledAt != nil {
		columns = append(columns, "last_scheduled_at")
		values = append(values, *n.LastScheduledAt)
	}
	if n.Ip != nil {
		columns = append(columns, "ip")
		values = append(values, n.Ip)
	}

	queryString, values, err := psq.Insert("nodes").Columns(columns...).Values(values...).
		Suffix("RETURNING *").ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := db.Get(n, queryString, values...); err != nil {
		return nil, utils.HandleDBError(err)
	}

	for _, label := range n.Labels {
		nodeLabel := &NodeLabel{
			NodeID: n.Id,
			Label:  label,
		}
		if err := nodeLabel.Create(db); err != nil {
			return nil, errors.Wrap(err, "nodeLabel.Create failed")
		}
	}
	nodeProto, err := n.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "node.ToProto failed")
	}
	return events.NewEvent(nodeProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil), nil
}

// Update performs an UPDATE SqlxGetter query for the Node fields given in the `updates` argument and returns a corresponding
// event.
//func (n *Node) Update(db *gorm.DB, updates map[string]interface{}) (*events_proto.Event, error) {
//	if n.ID == 0 {
//		return nil, errors.WithStack(status.Error(codes.FailedPrecondition, "can't update not saved Node"))
//	}
//	if err := utils.HandleDBError(db.Model(n).Updates(updates)); err != nil {
//		return nil, err
//	}
//	fieldMask := &field_mask.FieldMask{Paths: mapKeys(updates)}
//	nodeProto, err := n.ToProto(fieldMask)
//	if err != nil {
//		return nil, errors.Wrap(err, "node.ToProto failed")
//	}
//	event, err := events.NewEvent(nodeProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
//		fieldMask)
//	if err != nil {
//		return nil, err
//	}
//	return event, nil
//}

// LoadFromDB performs a lookup by ID and populates the struct.
func NewNodeFromDB(db utils.SqlxExtGetter, nodeId int64, forUpdate bool, logger *logrus.Logger) (*Node, error) {
	query := psq.Select("*").From("nodes").Where(sq.Eq{"id": nodeId})
	if forUpdate {
		query = query.Suffix("FOR UPDATE")
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var node Node
	if err := utils.HandleDBError(db.Get(&node, queryString, args...)); err != nil {
		return nil, err
	}
	if err := node.loadLabelsFromDB(db, logger); err != nil {
		return nil, errors.WithStack(err)
	}
	return &node, nil
}

// loadLabelsFromDB loads the corresponding NodeLabels to the Labels field.
func (n *Node) loadLabelsFromDB(db sqlx.Ext, logger *logrus.Logger) error {
	if n.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "Node is not saved in DB"))
	}
	query, args, err := psq.Select("label").From("node_labels").Where(sq.Eq{"node_id": n.Id}).ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	rows, err := db.Queryx(query, args...)
	if err != nil {
		return utils.HandleDBError(err)
	}
	defer utils.Close(rows, logger)

	n.Labels = nil
	var label string
	for rows.Next() {
		if err := rows.Scan(&label); err != nil {
			return errors.Wrap(err, "rows.Scan failed")
		}
		n.Labels = append(n.Labels, label)
	}
	return nil
}
//
//// AllocateContainerLimit allocates resources requested by the newLimit on the Node.
//// If the oldLimit provided, then the current Node's resources are adjusted accordingly.
//// This function returns a map that can be passed to the Node.Update() method.
//func (n *Node) AllocateContainerLimit(newLimit, oldLimit *Limit) (map[string]interface{}, error) {
//	if newLimit == nil {
//		return nil, errors.New("new limit can not be nil")
//	}
//	cpuAvailable := n.CpuAvailable
//	memoryAvailable := n.MemoryAvailable
//	diskAvailable := n.DiskAvailable
//	gpuAvailable := n.GpuAvailable
//
//	if oldLimit != nil {
//		cpuAvailable += oldLimit.Cpu
//		memoryAvailable += oldLimit.Memory
//		diskAvailable += oldLimit.Disk
//		gpuAvailable += oldLimit.Gpu
//	}
//
//	if cpuAvailable < newLimit.Cpu {
//		return nil, errors.
//			Errorf("failed to allocate Node CPU: %f requested, %f available", newLimit.Cpu, cpuAvailable)
//	}
//	if memoryAvailable < newLimit.Memory {
//		return nil, errors.
//			Errorf("failed to allocate Node memory: %d requested, %d available", newLimit.Memory, memoryAvailable)
//	}
//	if diskAvailable < newLimit.Disk {
//		return nil, errors.
//			Errorf("failed to allocate Node disk: %d requested, %d available", newLimit.Disk, diskAvailable)
//	}
//	if gpuAvailable < newLimit.Gpu {
//		return nil, errors.
//			Errorf("failed to allocate Node GPU: %d requested, %d available", newLimit.Gpu, gpuAvailable)
//	}
//	nodeUpdates := map[string]interface{}{
//		"cpu_available":    cpuAvailable - newLimit.Cpu,
//		"memory_available": memoryAvailable - newLimit.Memory,
//		"disk_available":   diskAvailable - newLimit.Disk,
//		"gpu_available":    gpuAvailable - newLimit.Gpu,
//	}
//	return nodeUpdates, nil
//}
//
//// DeallocateContainerLimit deallocates resources requested by the limit on the Node.
//// This function returns a map that can be passed to the Node.Update() method.
//func (n *Node) DeallocateContainerLimit(limit *Limit) map[string]interface{} {
//	nodeUpdates := map[string]interface{}{
//		"cpu_available":    n.CpuAvailable + limit.Cpu,
//		"memory_available": n.MemoryAvailable + limit.Memory,
//		"disk_available":   n.DiskAvailable + limit.Disk,
//		"gpu_available":    n.GpuAvailable + limit.Gpu,
//	}
//
//	return nodeUpdates
//}

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
//func (n *Node) PendingContainers(db *gorm.DB) ([]Container, error) {
//	q := db.Set("gorm:query_option", "FOR UPDATE").
//		Where("status = ?", utils.Enum(scheduler_proto.Container_PENDING)).
//		Where("? BETWEEN cpu_class_min AND cpu_class_max", n.CpuClass).
//		Where("? BETWEEN gpu_class_min AND gpu_class_max", n.GpuClass).
//		Where("? BETWEEN disk_class_min AND disk_class_max", n.DiskClass).
//		Where("ARRAY[?] @> labels", n.Labels).
//		Joins("INNER JOIN (SELECT container_id, MAX(id) from container_limits WHERE status = ? AND cpu > 0 group by container_id) as t1 on (t1.container_id = containers.id)", utils.Enum(scheduler_proto.Limit_CONFIRMED))
//
//	q = q.Limit(ContainersScheduledForNodeQueryLimit)
//
//	var containers []Container
//	if err := utils.HandleDBError(q.Find(containers)); err != nil {
//		return nil, errors.Wrap(err, "failed to FindPendingForNode for the Node")
//	}
//	return containers, nil
//}
//
//// Nodes represents a collection of Nodes.
//type Nodes []Node
//
//// List selects Containers from DB by the given filtering request and populates the receiver.
//// Returns the total number of Containers that satisfy the criteria and an error.
//func (nodes *Nodes) List(db *gorm.DB, request *scheduler_proto.ListNodesRequest) (uint32, error) {
//	query := db.Model(&Node{})
//	ordering := request.GetOrdering()
//
//	var orderBySQL string
//	switch ordering {
//	case scheduler_proto.ListNodesRequest_CONNECTED_AT_DESC:
//		orderBySQL = "connected_at DESC"
//	case scheduler_proto.ListNodesRequest_CONNECTED_AT_ASC:
//		orderBySQL = "created_at"
//	case scheduler_proto.ListNodesRequest_DISCONNECTED_AT_DESC:
//		orderBySQL = "disconnected_at DESC"
//	case scheduler_proto.ListNodesRequest_DISCONNECTED_AT_ASC:
//		orderBySQL = "disconnected_at"
//	case scheduler_proto.ListNodesRequest_SCHEDULED_AT_DESC:
//		orderBySQL = "scheduled_at DESC"
//	case scheduler_proto.ListNodesRequest_SCHEDULED_AT_ASC:
//		orderBySQL = "scheduled_at"
//	}
//	query = query.Order(orderBySQL)
//
//	// Filter by status.
//	if len(request.Status) > 0 {
//		enumStatus := make([]utils.Enum, len(request.Status))
//		for i, s := range request.Status {
//			enumStatus[i] = utils.Enum(s)
//		}
//		query = query.Where("status IN (?)", enumStatus)
//	}
//
//	if request.CpuAvailable > 0 {
//		query = query.Where("cpu_available >= ?", request.CpuAvailable)
//	}
//	if request.CpuClass != scheduler_proto.CPUClass_CPU_CLASS_UNKNOWN {
//		enum := utils.Enum(request.CpuClass)
//		query = query.Where("cpu_class_min <= ? AND cpu_class >= ?", enum, enum)
//	}
//
//	if request.MemoryAvailable > 0 {
//		query = query.Where("memory_available >= ?", request.MemoryAvailable)
//	}
//
//	if request.GpuAvailable > 0 {
//		query = query.Where("gpu_available >= ?", request.GpuAvailable)
//	}
//	if request.GpuClass != scheduler_proto.GPUClass_GPU_CLASS_UNKNOWN {
//		enum := utils.Enum(request.GpuClass)
//		query = query.Where("gpu_class_min <= ? AND gpu_class >= ?", enum, enum)
//	}
//
//	if request.DiskAvailable > 0 {
//		query = query.Where("disk_available >= ?", request.DiskAvailable)
//	}
//	if request.DiskClass != scheduler_proto.DiskClass_DISK_CLASS_UNKNOWN {
//		enum := utils.Enum(request.DiskClass)
//		query = query.Where("disk_class_min <= ? AND disk_class >= ?", enum, enum)
//	}
//
//	// TODO: add labels constraints.
//
//	if request.ContainersFinished > 0 {
//		query = query.Where("container_finished >= ?", request.ContainersFinished)
//	}
//
//	if request.ContainersFailed > 0 {
//		query = query.Where("container_failed <= ?", request.ContainersFailed)
//	}
//
//	// Perform a COUNT query with no limit and offset applied.
//	var count uint32
//	if err := utils.HandleDBError(query.Count(&count)); err != nil {
//		return 0, err
//	}
//
//	// Apply offset.
//	query = query.Offset(request.GetOffset())
//
//	// Apply limit.
//	var limit uint32
//	if request.Limit <= ListNodesMaxLimit {
//		limit = request.Limit
//	} else {
//		limit = ListNodesMaxLimit
//	}
//	query = query.Limit(limit)
//
//	// Perform a SELECT query.
//	if err := utils.HandleDBError(query.Find(nodes)); err != nil {
//		return 0, err
//	}
//	return count, nil
//}

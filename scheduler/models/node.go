package models

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// Create inserts a new Node in DB (also including its labels).
func (n *Node) Create(db utils.SqlxExtGetter) error {
	data := map[string]interface{}{
		"username":                 n.Username,
		"name":                     n.Name,
		"status":                   n.Status,
		"cpu_capacity":             n.CpuCapacity,
		"cpu_available":            n.CpuAvailable,
		"cpu_class":                n.CpuClass,
		"memory_capacity":          n.MemoryCapacity,
		"memory_available":         n.MemoryAvailable,
		"gpu_capacity":             n.GpuCapacity,
		"gpu_available":            n.GpuAvailable,
		"gpu_class":                n.GpuClass,
		"disk_capacity":            n.DiskCapacity,
		"disk_available":           n.DiskAvailable,
		"disk_class":               n.DiskClass,
		"network_ingress_capacity": n.NetworkIngressCapacity,
		"network_egress_capacity":  n.NetworkEgressCapacity,
		"containers_succeeded":     n.ContainersSucceeded,
		"containers_failed":        n.ContainersFailed,
		"containers_scheduled":     n.ContainersScheduled,
		"fingerprint":              n.Fingerprint,
	}

	// Nullable columns are used only when they have a value.
	if n.ConnectedAt != nil {
		data["connected_at"] = *n.ConnectedAt
	}
	if n.DisconnectedAt != nil {
		data["disconnected_at"] = *n.DisconnectedAt
	}
	if n.Ip != nil {
		data["ip"] = n.Ip
	}

	queryString, values, err := psq.Insert("nodes").SetMap(data).Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := db.Get(n, queryString, values...); err != nil {
		return utils.HandleDBError(err)
	}

	for _, label := range n.Labels {
		nodeLabel := &NodeLabel{
			NodeID: n.Id,
			Label:  label,
		}
		if err := nodeLabel.Create(db); err != nil {
			return errors.Wrap(err, "nodeLabel.Create failed")
		}
	}
	return nil
}

// Update performs an UPDATE query for the Node fields given in the `updates` argument and returns a corresponding
// event.
func (n *Node) Update(db utils.SqlxGetter, updates map[string]interface{}) error {
	if n.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't update not saved Node"))
	}

	queryString, args, err := psq.Update("nodes").Where(sq.Eq{"id": n.Id}).SetMap(updates).
		Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	return utils.HandleDBError(db.Get(n, queryString, args...))
}

// NewNodeFromDB performs a lookup by ID and returns a Node instance.
func NewNodeFromDB(db utils.SqlxExtGetter, nodeId int64) (*Node, error) {
	query := psq.Select("*").From("nodes").Where(sq.Eq{"id": nodeId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var node Node
	if err := utils.HandleDBError(db.Get(&node, queryString, args...)); err != nil {
		return nil, err
	}
	if err := node.loadLabelsFromDB(db); err != nil {
		return nil, errors.WithStack(err)
	}
	return &node, nil
}

// loadLabelsFromDB loads the corresponding NodeLabels to the Labels field.
func (n *Node) loadLabelsFromDB(db utils.SqlxSelector) error {
	if n.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "Node is not saved in DB"))
	}
	query, args, err := psq.Select("label").From("node_labels").Where(sq.Eq{"node_id": n.Id}).ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	return utils.HandleDBError(db.Select(&n.Labels, query, args...))
}

// GetCurrentPricing gets the current NodePricing policy for the Node.
func (n *Node) GetCurrentPricing(db utils.SqlxGetter) (*NodePricing, error) {
	query, args, err := psq.Select("*").From("node_pricing").Where("node_id = ?", n.Id).
		OrderBy("created_at DESC").Limit(1).ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var nodePricing NodePricing
	if err := db.Get(&nodePricing, query, args...); err != nil {
		return nil, utils.HandleDBError(err)
	}
	return &nodePricing, nil
}

// AllocateResourcesUpdates allocates resources requested by the newRequest on the Node.
// If the currentRequest provided, then the current Node's resources are adjusted accordingly.
// This function returns a map that can be passed to the Node.Update() method.
func (n *Node) AllocateResourcesUpdates(newRequest, currentRequest *ResourceRequest) (map[string]interface{}, error) {
	if newRequest == nil {
		return nil, errors.New("new limit can not be nil")
	}
	cpuAvailable := n.CpuAvailable
	memoryAvailable := n.MemoryAvailable
	diskAvailable := n.DiskAvailable
	gpuAvailable := n.GpuAvailable

	if currentRequest != nil {
		cpuAvailable += currentRequest.Cpu
		memoryAvailable += currentRequest.Memory
		diskAvailable += currentRequest.Disk
		gpuAvailable += currentRequest.Gpu
	}

	if cpuAvailable < newRequest.Cpu {
		return nil, status.Errorf(codes.ResourceExhausted,
			"failed to allocate Node CPU: %f requested, %f available", newRequest.Cpu, cpuAvailable)
	}
	if memoryAvailable < newRequest.Memory {
		return nil, status.Errorf(codes.ResourceExhausted,
			"failed to allocate Node memory: %d requested, %d available", newRequest.Memory, memoryAvailable)
	}
	if diskAvailable < newRequest.Disk {
		return nil, status.Errorf(codes.ResourceExhausted,
			"failed to allocate Node disk: %d requested, %d available", newRequest.Disk, diskAvailable)
	}
	if gpuAvailable < newRequest.Gpu {
		return nil, status.Errorf(codes.ResourceExhausted,
			"failed to allocate Node GPU: %d requested, %d available", newRequest.Gpu, gpuAvailable)
	}
	// TODO: rewrite without using absolute values.
	nodeUpdates := map[string]interface{}{
		"cpu_available":    cpuAvailable - newRequest.Cpu,
		"memory_available": memoryAvailable - newRequest.Memory,
		"disk_available":   diskAvailable - newRequest.Disk,
		"gpu_available":    gpuAvailable - newRequest.Gpu,
	}
	return nodeUpdates, nil
}

// DeallocateResourcesUpdates updates the Nodes resources and returns a map to be passed to Node.Update().
func (n *Node) DeallocateResourcesUpdates(request *ResourceRequest) map[string]interface{} {
	return map[string]interface{}{
		"cpu_available":    sq.Expr("cpu_available + ?", request.Cpu),
		"gpu_available":    sq.Expr("gpu_available + ?", request.Gpu),
		"memory_available": sq.Expr("memory_available + ?", request.Memory),
		"disk_available":   sq.Expr("disk_available + ?", request.Disk),
	}
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
//func (n *Node) PendingContainers(db *gorm.DB) ([]Container, error) {
//	q := db.Set("gorm:query_option", "FOR UPDATE").
//		Where("status = ?", utils.Enum(scheduler_proto.Container_PENDING)).
//		Where("? BETWEEN cpu_class_min AND cpu_class_max", n.CpuClass).
//		Where("? BETWEEN gpu_class_min AND gpu_class_max", n.GpuClass).
//		Where("? BETWEEN disk_class_min AND disk_class_max", n.DiskClass).
//		Where("ARRAY[?] @> labels", n.Labels).
//		Joins("INNER JOIN (SELECT container_id, MAX(id) from container_limits WHERE status = ? AND cpu > 0 group by container_id) as t1 on (t1.container_id = containers.id)", utils.Enum(scheduler_proto.Limit_CONFIRMED))
//
//	q = q.ResourceRequest(ContainersScheduledForNodeQueryLimit)
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


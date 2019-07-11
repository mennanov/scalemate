package models

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// ContainersScheduledForNodeQueryLimit is the number of Containers to be selected from DB when Containers are
	// selected to be scheduled on a Node.
	ContainersScheduledForNodeQueryLimit = 100
	// ListContainersMaxLimit is a maximum allowed limit in the SqlxGetter query that lists Containers.
	ListContainersMaxLimit uint32 = 300
)

var psq = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// Container defines a Container model.
type Container struct {
	scheduler_proto.Container
}

// NewContainerFromDB performs a lookup by ID and populates the struct.
func NewContainerFromDB(db utils.SqlxExtGetter, containerId int64) (*Container, error) {
	query := psq.Select("*").From("containers").Where(sq.Eq{"id": containerId})

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var container Container
	if err := utils.HandleDBError(db.Get(&container, queryString, args...)); err != nil {
		return nil, err
	}
	if err := container.loadLabelsFromDB(db); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := container.loadSchedulingStrategiesFromDB(db); err != nil {
		return nil, errors.WithStack(err)
	}
	return &container, nil
}

// ToProto returns the corresponding scheduler_proto.Container instance with the mask applied.
func (c *Container) ToProto(paths []string) (*scheduler_proto.Container, error) {
	if len(paths) == 0 {
		return &c.Container, nil
	}

	mask, err := fieldmask_utils.MaskFromPaths(paths, generator.CamelCase)
	if err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
	}
	// Always include Container ID regardless of the mask.
	protoFiltered := &scheduler_proto.Container{Id: c.Id}
	if err := fieldmask_utils.StructToStruct(mask, &c.Container, protoFiltered); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
	}
	return protoFiltered, nil
}

// ToProtoFromUpdates returns a scheduler_proto.Container with the applied FieldMask derived from the given paths.
func (c *Container) ToProtoFromUpdates(updates map[string]interface{}) (*scheduler_proto.Container, *types.FieldMask, error) {
	paths := utils.MapKeys(updates)
	containerProto, err := c.ToProto(paths)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	fieldMask := &types.FieldMask{Paths: paths}
	return containerProto, fieldMask, nil
}

// Create inserts a new Container in DB and returns the corresponding event.
func (c *Container) Create(db utils.SqlxGetter) error {
	data := map[string]interface{}{
		"username":            c.Username,
		"status":              c.Status,
		"status_message":      c.StatusMessage,
		"image":               c.Image,
		"node_auth_token":     c.NodeAuthToken,
		"cpu_class_min":       c.CpuClassMin,
		"cpu_class_max":       c.CpuClassMax,
		"gpu_class_min":       c.GpuClassMin,
		"gpu_class_max":       c.GpuClassMax,
		"disk_class_min":      c.DiskClassMin,
		"disk_class_max":      c.DiskClassMax,
		"network_ingress_min": c.NetworkIngressMin,
		"network_egress_min":  c.NetworkEgressMin,
		"max_price_limit":     c.MaxPriceLimit,
	}

	if c.Id != 0 {
		data["id"] = c.Id
	}
	if c.NodeId != nil {
		data["node_id"] = *c.NodeId
	}
	if c.NodePricingId != nil {
		data["node_pricing_id"] = *c.NodePricingId
	}
	queryString, args, err := psq.Insert("containers").SetMap(data).Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := db.Get(c, queryString, args...); err != nil {
		return utils.HandleDBError(err)
	}
	for _, label := range c.Labels {
		containerLabel := &ContainerLabel{
			ContainerId: c.Id,
			Label:       label,
		}
		if err := containerLabel.Create(db); err != nil {
			return errors.Wrap(err, "containerLabel.Create failed")
		}
	}
	for i, strategy := range c.SchedulingStrategy {
		containerSchedulingStrategy := &ContainerSchedulingStrategy{
			SchedulingStrategy: scheduler_proto.SchedulingStrategy{
				Strategy:             strategy.Strategy,
				DifferencePercentage: strategy.DifferencePercentage,
			},
			ContainerId: c.Id,
			Position:    i,
		}
		if err := containerSchedulingStrategy.Create(db); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// loadLabelsFromDB loads the corresponding ContainerLabels to the Labels field.
func (c *Container) loadLabelsFromDB(db utils.SqlxSelector) error {
	if c.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "Container is not saved in DB"))
	}
	query, args, err := psq.Select("label").From("container_labels").Where(sq.Eq{"container_id": c.Id}).ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	return utils.HandleDBError(db.Select(&c.Labels, query, args...))
}

// loadSchedulingStrategiesFromDB loads the corresponding ContainerLabels to the Labels field.
func (c *Container) loadSchedulingStrategiesFromDB(db utils.SqlxSelector) error {
	if c.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "Container is not saved in DB"))
	}
	query, args, err := psq.Select("strategy, difference_percentage").From("container_scheduling_strategies").
		Where(sq.Eq{"container_id": c.Id}).OrderBy("position ASC").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	return utils.HandleDBError(db.Select(&c.SchedulingStrategy, query, args...))
}

// ContainerStatusTransitions defined possible Container status transitions.
var ContainerStatusTransitions = map[scheduler_proto.Container_Status][]scheduler_proto.Container_Status{
	scheduler_proto.Container_NEW: {
		scheduler_proto.Container_SCHEDULED,
		scheduler_proto.Container_STOPPED,
	},
	scheduler_proto.Container_SCHEDULED: {
		scheduler_proto.Container_STOPPED,
		scheduler_proto.Container_PREPARING,
		scheduler_proto.Container_OFFLINE,
	},
	scheduler_proto.Container_PREPARING: {
		scheduler_proto.Container_STOPPED,
		scheduler_proto.Container_RUNNING,
		scheduler_proto.Container_FAILED,
		scheduler_proto.Container_OFFLINE,
	},
	scheduler_proto.Container_RUNNING: {
		scheduler_proto.Container_STOPPED,
		scheduler_proto.Container_OFFLINE,
		scheduler_proto.Container_EVICTED,
	},
	scheduler_proto.Container_STOPPED: {},
	scheduler_proto.Container_FAILED:  {},
	scheduler_proto.Container_OFFLINE: {
		scheduler_proto.Container_STOPPED,
		scheduler_proto.Container_RUNNING,
		scheduler_proto.Container_NODE_FAILED,
		scheduler_proto.Container_EVICTED,
	},
	scheduler_proto.Container_NODE_FAILED: {},
	scheduler_proto.Container_EVICTED:     {},
}

// ValidateNewStatus checks whether the status of the Container can be changed to the given one.
func (c *Container) ValidateNewStatus(newStatus scheduler_proto.Container_Status) error {
	if c.Id == 0 {
		return status.Error(codes.FailedPrecondition, "can't update status for an unsaved Container")
	}
	// Validate the containerStatus transition.
	containerStatus := scheduler_proto.Container_Status(c.Status)
	for _, s := range ContainerStatusTransitions[containerStatus] {
		if s == newStatus {
			return nil
		}
	}
	return status.Errorf(codes.InvalidArgument, "container with status %s can't be updated to %s",
		containerStatus.String(), newStatus.String())
}

// Update performs an UPDATE query for the Container fields given in the `updates` argument.
func (c *Container) Update(db utils.SqlxGetter, updates map[string]interface{}) error {
	if c.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't update unsaved Container"))
	}
	query, args, err := psq.Update("containers").SetMap(updates).Where(sq.Eq{"id": c.Id}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	return utils.HandleDBError(db.Get(c, query, args...))
}

// isFinalContainerStatus return true if the given Container status is final (can't be changed).
func isFinalContainerStatus(containerStatus scheduler_proto.Container_Status) bool {
	return len(ContainerStatusTransitions[containerStatus]) == 0
}

// IsTerminated returns true if Container is in the final status (terminated).
func (c *Container) IsTerminated() bool {
	return isFinalContainerStatus(c.Status)
}

// NodesForScheduling finds the Nodes this Container can be scheduled on.
func (c *Container) NodesForScheduling(db utils.SqlxSelector, request *ResourceRequest) ([]*NodeExt, error) {
	query := psq.Select(
		fmt.Sprintf(`nodes.*, p.id AS pricing_id, cpu_price, gpu_price, memory_price, disk_price, 
		COUNT(*) AS containers_scheduled, SUM(CASE WHEN c.status = %d THEN 1 ELSE 0 END) AS containers_node_failed`,
			scheduler_proto.Container_NODE_FAILED)).
		From("nodes").Where(sq.Eq{"nodes.status": scheduler_proto.Node_ONLINE}).
		JoinClause("LEFT JOIN containers AS c ON(nodes.id = c.node_id)").
		GroupBy("nodes.id, p.id, p.cpu_price, p.gpu_price, p.memory_price, p.disk_price")

	pricingQuery := psq.Select("*").
		FromSelect(
			psq.Select("*, rank() OVER (PARTITION BY node_id ORDER BY created_at DESC) as position").
				From("node_pricing"), "pricing").
		Where("pricing.position = 1")

	query = query.JoinClause(pricingQuery.Prefix("INNER JOIN (").Suffix(") p ON (p.node_id = nodes.id)"))

	if request.Cpu != 0 {
		query = query.Where("cpu_available >= ?", request.Cpu)
	}
	if request.Gpu != 0 {
		query = query.Where("gpu_available >= ?", request.Gpu)
	}
	if request.Memory != 0 {
		query = query.Where("memory_available >= ?", request.Memory)
	}
	if request.Disk != 0 {
		query = query.Where("disk_available >= ?", request.Disk)
	}

	if c.NetworkEgressMin != 0 {
		query = query.Where("network_egress_capacity >= ?", c.NetworkEgressMin)
	}
	if c.NetworkIngressMin != 0 {
		query = query.Where("network_ingress_capacity >= ?", c.NetworkIngressMin)
	}
	if c.CpuClassMin != 0 {
		query = query.Where("cpu_class >= ?", c.CpuClassMin)
	}
	if c.CpuClassMax != 0 {
		query = query.Where("cpu_class <= ?", c.CpuClassMax)
	}
	if c.GpuClassMin != 0 {
		query = query.Where("gpu_class >= ?", c.GpuClassMin)
	}
	if c.GpuClassMax != 0 {
		query = query.Where("gpu_class <= ?", c.GpuClassMax)
	}
	if c.DiskClassMin != 0 {
		query = query.Where("disk_class >= ?", c.DiskClassMin)
	}
	if c.DiskClassMax != 0 {
		query = query.Where("disk_class <= ?", c.DiskClassMax)
	}
	if len(c.Labels) != 0 {
		query = query.Where(`0 < ALL (
		SELECT (
			SELECT COUNT(*) FROM node_labels AS nl WHERE nl.node_id = nodes.id AND nl.label ~* cl.label
		) FROM container_labels AS cl WHERE cl.container_id = ?)`, c.Id)
	}
	if c.MaxPriceLimit > 0 {
		query = query.Where("cpu_price * ? + gpu_price * ? + disk_price * ? + memory_price * ? <= ?",
			request.Cpu, request.Gpu, request.Disk, request.Memory, c.MaxPriceLimit)
	}

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var nodesWithPricing []*NodeExt
	if err := db.Select(&nodesWithPricing, queryString, args...); err != nil {
		return nil, utils.HandleDBError(err)
	}
	return nodesWithPricing, nil
}

// CurrentResourceRequest finds the current (most recently confirmed) Container's CurrentResourceRequest.
func (c *Container) CurrentResourceRequest(db utils.SqlxGetter) (*ResourceRequest, error) {
	query := psq.Select("*").From("resource_requests").
		Where("container_id = ? AND status = ?", c.Id, scheduler_proto.ResourceRequest_CONFIRMED).
		Limit(1).
		OrderBy("created_at DESC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var request ResourceRequest
	if err := utils.HandleDBError(db.Get(&request, queryString, args...)); err != nil {
		return nil, err
	}
	return &request, nil
}

// ResourceRequests finds all corresponding Container's ResourceRequests.
func (c *Container) ResourceRequests(db utils.SqlxSelector) ([]*ResourceRequest, error) {
	query := psq.Select("*").From("resource_requests").
		Where("container_id = ?", c.Id).OrderBy("created_at ASC")

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var requests []*ResourceRequest
	if err := utils.HandleDBError(db.Select(&requests, queryString, args...)); err != nil {
		return nil, err
	}
	return requests, nil
}

// Terminate validates and updates the Container's status, deallocates resources on the corresponding Node.
func (c *Container) Terminate(db utils.SqlxGetter, newStatus scheduler_proto.Container_Status) ([]*events_proto.Event, error) {
	if c.NodeId == nil {
		return nil, status.Error(codes.FailedPrecondition, "Container has not been scheduled")
	}
	if !isFinalContainerStatus(newStatus) {
		return nil, status.Errorf(codes.InvalidArgument, "newStatus status %s is not final", newStatus.String())
	}
	if c.IsTerminated() {
		return nil, status.Error(codes.FailedPrecondition, "Container is already terminated")
	}
	if err := c.ValidateNewStatus(newStatus); err != nil {
		return nil, errors.WithStack(err)
	}

	resourceRequest, err := c.CurrentResourceRequest(db)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	node := &Node{scheduler_proto.Node{Id: *c.NodeId}}
	updates := node.DeallocateResourcesUpdates(resourceRequest)

	err = node.Update(db, updates)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = c.Update(db, map[string]interface{}{"status": newStatus})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	//return []*events_proto.Event{nodeUpdatedEvent, containerUpdatedEvent}, nil
	return nil, nil
}

// ListContainers selects Containers from DB by the given filtering request.
// Returns the Containers slice, a total number of Containers that satisfy the criteria and an error.
func ListContainers(db utils.SqlxExtGetter, username string, request *scheduler_proto.ListContainersRequest) ([]Container, uint32, error) {
	query := psq.Select().From("containers").Where(sq.Eq{"username": username})

	if len(request.Status) != 0 {
		query = query.Where(sq.Eq{"status": request.Status})
	}

	var count uint32
	if err := query.Columns("COUNT(*)").RunWith(db).Scan(&count); err != nil {
		return nil, 0, utils.HandleDBError(err)
	}

	if count == 0 {
		// If the count is already 0 don't bother running the subsequent SELECT query.
		return nil, 0, status.Error(codes.NotFound, "no containers found")
	}

	limit := ListContainersMaxLimit
	if request.Limit <= ListContainersMaxLimit {
		limit = request.Limit
	}

	var orderBy string
	switch request.Ordering {
	case scheduler_proto.ListContainersRequest_CREATED_AT_ASC:
		orderBy = "created_at"
	case scheduler_proto.ListContainersRequest_UPDATED_AT_ASC:
		orderBy = "updated_at"
	case scheduler_proto.ListContainersRequest_UPDATED_AT_DESC:
		orderBy = "updated_at DESC"
	default:
		orderBy = "created_at DESC"
	}

	query = query.Columns("*").Limit(uint64(limit)).Offset(uint64(request.Offset)).OrderBy(orderBy)

	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}

	var containers []Container
	if err := db.Select(&containers, queryString, args...); err != nil {
		return nil, 0, utils.HandleDBError(err)
	}

	return containers, count, nil
}

// NewContainerFromProto creates a new instance of Container populated from the given proto message.
func NewContainerFromProto(p *scheduler_proto.Container) *Container {
	return &Container{Container: *p}
}

// ContainerLabel is a Container's label.
type ContainerLabel struct {
	ContainerId int64
	Label       string
}

// Create inserts a new ContainerLabel in DB.
func (c *ContainerLabel) Create(db utils.SqlxGetter) error {
	queryString, args, err := psq.Insert("container_labels").
		Columns("container_id", "label").
		Values(c.ContainerId, c.Label).
		Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}
	return utils.HandleDBError(db.Get(c, queryString, args...))
}

// ContainerSchedulingStrategy is a Container's scheduling strategy model.
type ContainerSchedulingStrategy struct {
	scheduler_proto.SchedulingStrategy
	ContainerId int64
	Position    int
}

// Create inserts a new ContainerSchedulingStrategy in DB.
func (c *ContainerSchedulingStrategy) Create(db utils.SqlxGetter) error {
	queryString, args, err := psq.Insert("container_scheduling_strategies").SetMap(map[string]interface{}{
		"container_id":          c.ContainerId,
		"position":              c.Position,
		"strategy":              c.Strategy,
		"difference_percentage": c.DifferencePercentage,
	}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}
	return utils.HandleDBError(db.Get(c, queryString, args...))
}

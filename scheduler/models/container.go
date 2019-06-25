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

	"github.com/mennanov/scalemate/shared/events"
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
func NewContainerFromDB(db utils.SqlxExtGetter, containerId int64, forUpdate bool) (*Container, error) {
	query := psq.Select("*").From("containers").Where(sq.Eq{"id": containerId})
	if forUpdate {
		query = query.Suffix("FOR UPDATE")
	}

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
	return &container, nil
}

// ToProto populates the given `*scheduler_proto.Container` with the contents of the Container.
func (c *Container) ToProto(fieldMask *types.FieldMask) (*scheduler_proto.Container, error) {
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		return &c.Container, nil
	}

	mask, err := fieldmask_utils.MaskFromPaths(fieldMask.Paths, generator.CamelCase)
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

// Create inserts a new Container in DB and returns the corresponding event.
func (c *Container) Create(db utils.SqlxGetter) (*events_proto.Event, error) {
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
		return nil, errors.WithStack(err)
	}

	if err := db.Get(c, queryString, args...); err != nil {
		return nil, utils.HandleDBError(err)
	}
	for _, label := range c.Labels {
		containerLabel := &ContainerLabel{
			ContainerId: c.Id,
			Label:       label,
		}
		if err := containerLabel.Create(db); err != nil {
			return nil, errors.Wrap(err, "containerLabel.Create failed")
		}
	}
	containerProto, err := c.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "container.ToProto failed")
	}
	return events.NewEvent(containerProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil), nil
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
		scheduler_proto.Container_OFFLINE_FAILED,
		scheduler_proto.Container_EVICTED,
	},
	scheduler_proto.Container_OFFLINE_FAILED: {},
	scheduler_proto.Container_EVICTED:        {},
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
	return status.Errorf(codes.FailedPrecondition, "container with status %s can't be updated to %s",
		containerStatus.String(), newStatus.String())
}

// Update performs an UPDATE query for the Container fields given in the `updates` argument and returns a
// corresponding event.
func (c *Container) Update(db utils.SqlxGetter, updates map[string]interface{}) (*events_proto.Event, error) {
	if c.Id == 0 {
		return nil, errors.WithStack(status.Error(codes.FailedPrecondition, "can't update unsaved Container"))
	}
	query, args, err := psq.Update("containers").SetMap(updates).Where(sq.Eq{"id": c.Id}).Suffix("RETURNING *").ToSql()
	if err := db.Get(c, query, args...); err != nil {
		return nil, utils.HandleDBError(err)
	}

	fieldMask := &types.FieldMask{Paths: mapKeys(updates)}
	containerProto, err := c.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "container.ToProto failed")
	}
	return events.NewEvent(containerProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER, fieldMask), nil
}

// isFinalContainerStatus return true if the given Container status is final (can't be changed).
func isFinalContainerStatus(containerStatus scheduler_proto.Container_Status) bool {
	return len(ContainerStatusTransitions[containerStatus]) == 0
}

// IsTerminated returns true if Container is in the final status (terminated).
func (c *Container) IsTerminated() bool {
	return isFinalContainerStatus(c.Status)
}

// Schedule finds the Node that satisfies the Container's criteria and the given ResourceRequest, updates the Node's
// resources (*_available field values) and sets the status of the Container to SCHEDULED.
func (c *Container) Schedule(db utils.SqlxGetter, request *ResourceRequest) ([]*events_proto.Event, error) {
	if c.NodeId != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Container has already been scheduled")
	}
	lookupQuery := psq.Select("id").From("nodes").Where(sq.Eq{"status": scheduler_proto.Node_ONLINE}).Limit(1)

	if request.Cpu != 0 {
		lookupQuery = lookupQuery.Where("cpu_available >= ?", request.Cpu)
	}
	if request.Gpu != 0 {
		lookupQuery = lookupQuery.Where("gpu_available >= ?", request.Gpu)
	}
	if request.Memory != 0 {
		lookupQuery = lookupQuery.Where("memory_available >= ?", request.Memory)
	}
	if request.Disk != 0 {
		lookupQuery = lookupQuery.Where("disk_available >= ?", request.Disk)
	}

	if c.NetworkEgressMin != 0 {
		lookupQuery = lookupQuery.Where("network_egress_capacity >= ?", c.NetworkEgressMin)
	}
	if c.NetworkIngressMin != 0 {
		lookupQuery = lookupQuery.Where("network_ingress_capacity >= ?", c.NetworkIngressMin)
	}
	if c.CpuClassMin != 0 {
		lookupQuery = lookupQuery.Where("cpu_class >= ?", c.CpuClassMin)
	}
	if c.CpuClassMax != 0 {
		lookupQuery = lookupQuery.Where("cpu_class <= ?", c.CpuClassMax)
	}
	if c.GpuClassMin != 0 {
		lookupQuery = lookupQuery.Where("gpu_class >= ?", c.GpuClassMin)
	}
	if c.GpuClassMax != 0 {
		lookupQuery = lookupQuery.Where("gpu_class <= ?", c.GpuClassMax)
	}
	if c.DiskClassMin != 0 {
		lookupQuery = lookupQuery.Where("disk_class >= ?", c.DiskClassMin)
	}
	if c.DiskClassMax != 0 {
		lookupQuery = lookupQuery.Where("disk_class <= ?", c.DiskClassMax)
	}
	if len(c.Labels) != 0 {
		lookupQuery = lookupQuery.Where(`0 < ALL (
		SELECT (
			SELECT COUNT(*) FROM node_labels AS nl WHERE nl.node_id = nodes.id AND nl.label ~* cl.label
		) FROM container_labels AS cl WHERE cl.container_id = ?)`, c.Id)
	}

	switch c.SchedulingStrategy {
	case scheduler_proto.Container_CHEAPEST:
		pricingQuery := psq.Select("node_id, cpu_price, gpu_price, memory_price, disk_price").
			FromSelect(
				psq.Select("*, rank() OVER (PARTITION BY node_id ORDER BY created_at DESC) as pos").
					From("node_pricing"), "pricing").
			Where("pricing.pos = 1")

		lookupQuery = lookupQuery.JoinClause(pricingQuery.Prefix("JOIN (").Suffix(") p ON (p.node_id = nodes.id)")).
			OrderBy(fmt.Sprintf("cpu_price * %d + gpu_price * %d + memory_price * %d + disk_price * %d",
				request.Cpu, request.Gpu, request.Memory, request.Disk))

	case scheduler_proto.Container_LEAST_BUSY:
		lookupQuery = lookupQuery.OrderBy(`CASE WHEN gpu_capacity > 0 THEN
		(cpu_available::float / cpu_capacity::float)^2 + (gpu_available::float / gpu_capacity::float)^2 +
		(disk_available::float / disk_capacity::float)^2 + (memory_available::float / memory_capacity::float)^2
		ELSE
		(cpu_available::float / cpu_capacity::float)^2 +
		(disk_available::float / disk_capacity::float)^2 + (memory_available::float / memory_capacity::float)^2
		END`)
	case scheduler_proto.Container_MOST_RELIABLE:
		lookupQuery = lookupQuery.OrderBy(`CASE
		WHEN containers_finished == 0
			THEN Infinity
		ELSE containers_failed::float / containers_finished::float
		END`)
	}

	updateQuery := psq.Update("nodes").Where(lookupQuery.Prefix("id IN(").Suffix(")")).Suffix("RETURNING *")

	nodeUpdates := map[string]interface{}{
		"cpu_available":     sq.Expr("cpu_available - ?", request.Cpu),
		"gpu_available":     sq.Expr("gpu_available - ?", request.Gpu),
		"memory_available":  sq.Expr("memory_available - ?", request.Memory),
		"disk_available":    sq.Expr("disk_available - ?", request.Disk),
		"last_scheduled_at": sq.Expr("NOW()"),
	}
	updateQuery = updateQuery.SetMap(nodeUpdates)

	queryString, args, err := updateQuery.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var node Node
	if err := db.Get(&node, queryString, args...); err != nil {
		return nil, utils.HandleDBError(err)
	}
	nodePricing, err := node.GetCurrentPricing(db)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fieldMask := &types.FieldMask{Paths: mapKeys(nodeUpdates)}
	nodeProto, err := (&node).ToProto(fieldMask)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	nodeEvent := events.NewEvent(nodeProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER, fieldMask)

	containerEvent, err := c.Update(db, map[string]interface{}{
		"status":          scheduler_proto.Container_SCHEDULED,
		"node_id":         node.Id,
		"node_pricing_id": nodePricing.Id,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return []*events_proto.Event{nodeEvent, containerEvent}, nil
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

	resources, err := NewResourceRequestFromDBLatest(db, c.Id)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	node := &Node{scheduler_proto.Node{Id: *c.NodeId}}
	updates := node.DeallocateResources(resources)
	if newStatus == scheduler_proto.Container_OFFLINE_FAILED {
		updates["containers_failed"] = sq.Expr("containers_failed + 1")
	}
	updates["containers_finished"] = sq.Expr("containers_finished + 1")

	nodeUpdatedEvent, err := node.Update(db, updates)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	containerUpdatedEvent, err := c.Update(db, map[string]interface{}{"status": newStatus})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return []*events_proto.Event{nodeUpdatedEvent, containerUpdatedEvent}, nil
}

// ListContainers selects Containers from DB by the given filtering request.
// Returns the Containers slice, a total number of Containers that satisfy the criteria and an error.
func ListContainers(db utils.SqlxExtGetter, request *scheduler_proto.ListContainersRequest) ([]Container, uint32, error) {
	query := psq.Select().From("containers").Where(sq.Eq{"username": request.Username})

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

// NewContainerFromProto creates a new instance of Container populated from the given proto message.
func NewContainerFromProto(p *scheduler_proto.Container) *Container {
	return &Container{Container: *p}
}

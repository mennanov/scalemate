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

// LoadFromDB performs a lookup by ID and populates the struct.
func NewContainerFromDB(db utils.SqlxExtGetter, containerId int64, forUpdate bool, logger *logrus.Logger) (*Container, error) {
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
	if err := container.loadLabelsFromDB(db, logger); err != nil {
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
func (c *Container) Create(db utils.SqlxExtGetter) (*events_proto.Event, error) {
	// NOT NULL columns are always used.
	columns := []string{
		"username",
		"status", "status_message",
		"image",
		"agent_auth_token",
		"cpu_class_min", "cpu_class_max",
		"gpu_class_min", "gpu_class_max",
		"disk_class_min", "disk_class_max",
		"network_ingress_min", "network_egress_min",
	}
	values := []interface{}{
		c.Username,
		int32(c.Status), c.StatusMessage,
		c.Image,
		c.AgentAuthToken,
		c.CpuClassMin, c.CpuClassMax,
		c.GpuClassMin, c.GpuClassMax,
		c.DiskClassMin, c.DiskClassMax,
		c.NetworkIngressMin, c.NetworkEgressMin,
	}
	if c.Id != 0 {
		columns = append(columns, "id")
		values = append(values, c.Id)
	}
	if c.NodeId != nil {
		columns = append(columns, "node_id")
		values = append(values, *c.NodeId)
	}
	queryString, args, err := psq.Insert("containers").Columns(columns...).Values(values...).
		Suffix("RETURNING *").ToSql()
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
func (c *Container) loadLabelsFromDB(db sqlx.Ext, logger *logrus.Logger) error {
	if c.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "Container is not saved in DB"))
	}
	query, args, err := psq.Select("label").From("container_labels").Where(sq.Eq{"container_id": c.Id}).ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	rows, err := db.Queryx(query, args...)
	if err != nil {
		return utils.HandleDBError(err)
	}
	defer utils.Close(rows, logger)

	c.Labels = nil
	var label string
	for rows.Next() {
		if err := rows.Scan(&label); err != nil {
			return errors.Wrap(err, "rows.Scan failed")
		}
		c.Labels = append(c.Labels, label)
	}
	return nil
}

// ContainerStatusTransitions defined possible Container status transitions.
var ContainerStatusTransitions = map[scheduler_proto.Container_Status][]scheduler_proto.Container_Status{
	scheduler_proto.Container_NEW: {
		scheduler_proto.Container_DECLINED,
		scheduler_proto.Container_PENDING,
		scheduler_proto.Container_CANCELLED,
	},
	scheduler_proto.Container_DECLINED: {},
	scheduler_proto.Container_PENDING: {
		scheduler_proto.Container_CANCELLED,
		scheduler_proto.Container_SCHEDULED,
	},
	scheduler_proto.Container_CANCELLED: {},
	scheduler_proto.Container_SCHEDULED: {
		scheduler_proto.Container_RUNNING,
		scheduler_proto.Container_FAILED,
	},
	scheduler_proto.Container_RUNNING: {
		scheduler_proto.Container_STOPPED,
	},
	scheduler_proto.Container_STOPPED: {},
	scheduler_proto.Container_FAILED:  {},
}

// UpdateStatus updates the Container's status.
func (c *Container) UpdateStatus(
	db utils.SqlxGetter,
	newStatus scheduler_proto.Container_Status,
	logger *logrus.Logger,
) (*events_proto.Event, error) {
	if c.Id == 0 {
		return nil, status.Error(codes.FailedPrecondition, "can't update status for an unsaved Container")
	}
	// Validate the containerStatus transition.
	containerStatus := scheduler_proto.Container_Status(c.Status)
	for _, s := range ContainerStatusTransitions[containerStatus] {
		if s == newStatus {
			return c.Update(db, map[string]interface{}{
				"status": newStatus,
			}, logger)
		}
	}
	return nil, status.Errorf(codes.FailedPrecondition, "container with status %s can't be updated to %s",
		containerStatus.String(), newStatus.String())
}

// Update performs an UPDATE SqlxGetter query for the Container fields given in the `updates` argument and returns a
// corresponding event.
func (c *Container) Update(
	db utils.SqlxGetter,
	updates map[string]interface{},
	logger *logrus.Logger,
) (*events_proto.Event, error) {
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
		return nil, errors.Wrap(err, "job.ToProto failed")
	}
	return events.NewEvent(containerProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER, fieldMask), nil
}

// CurrentLimit gets the most recently confirmed Limit for this Container.
//func (c *Container) CurrentLimit(db *gorm.DB) (*Limit, error) {
//	limit := &Limit{
//		Container:   c,
//		ContainerID: c.ID,
//		Status:      utils.Enum(scheduler_proto.Limit_CONFIRMED),
//	}
//	if err := utils.HandleDBError(db.Where(limit).Order("id DESC").First(limit)); err != nil {
//		return nil, errors.Wrap(err, "failed to find Limit")
//	}
//	return limit, nil
//}

//// NodesAvailable finds the best available Node to schedule this Container on.
//func (c *Container) NodesAvailable(db *gorm.DB) ([]*Node, error) {
//	if c.NodeID != nil {
//		return nil, status.Error(codes.FailedPrecondition, "Container is already scheduled")
//	}
//	q := db.Set("gorm:query_option", "FOR UPDATE OF nodes").
//		Table("nodes").
//		Where("status = ?", utils.Enum(scheduler_proto.Node_ONLINE)).
//		Where("cpu_class BETWEEN ? AND ?", c.CpuClassMin, c.CpuClassMax).
//		Where("gpu_class BETWEEN ? AND ?", c.GpuClassMin, c.GpuClassMax).
//		Where("disk_class BETWEEN ? AND ?", c.DiskClassMin, c.DiskClassMax).
//		// FIXME: add labels constraints.
//		Where("")
//
//	var nodes []*Node
//	if err := utils.HandleDBError(q.Find(&nodes)); err != nil {
//		return nil, errors.Wrap(err, "failed to find Nodes")
//	}
//
//	return nodes, nil
//}
//
//// Schedule allocates resources on the given Node and sets the Container status to SCHEDULED.
//func (c *Container) Schedule(db *gorm.DB, node *Node) ([]*events_proto.Event, error) {
//	if c.ID == 0 {
//		return nil, status.Error(codes.FailedPrecondition, "can't schedule unsaved Container")
//	}
//	if c.NodeID != nil {
//		return nil, status.Error(codes.FailedPrecondition, "Container is already scheduled")
//	}
//
//	currentLimit, err := c.CurrentLimit(db)
//	if err != nil {
//		return nil, errors.Wrap(err, "container.CurrentLimit failed")
//	}
//
//	allocationUpdates, err := node.AllocateContainerLimit(currentLimit, nil)
//	if err != nil {
//		return nil, errors.Wrap(err, "failed to allocate Node resources")
//	}
//	now := time.Now().UTC()
//	allocationUpdates["scheduled_at"] = &now
//	nodeUpdatedEvent, err := node.Update(db, allocationUpdates)
//	if err != nil {
//		return nil, errors.Wrap(err, "node.Update failed")
//	}
//
//	// Update the Container status to SCHEDULED.
//	containerUpdatedEvent, err := c.UpdateStatus(db, scheduler_proto.Container_SCHEDULED)
//	if err != nil {
//		return nil, errors.Wrap(err, "failed to update Container status")
//	}
//
//	return []*events_proto.Event{nodeUpdatedEvent, containerUpdatedEvent}, nil
//}

// IsTerminated returns true if Container is in the final status (terminated).
func (c *Container) IsTerminated() bool {
	return c.Status == scheduler_proto.Container_STOPPED ||
		c.Status == scheduler_proto.Container_FAILED ||
		c.Status == scheduler_proto.Container_CANCELLED ||
		c.Status == scheduler_proto.Container_DECLINED
}

// ListContainers selects Containers from DB by the given filtering request.
// Returns the Containers slice, a total number of Containers that satisfy the criteria and an error.
func ListContainers(db *sqlx.DB, request *scheduler_proto.ListContainersRequest) ([]Container, uint32, error) {
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
		return []Container{}, 0, nil
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
func (c *ContainerLabel) Create(db sqlx.Execer) error {
	_, err := db.Exec("INSERT INTO container_labels (container_id, label) VALUES ($1, $2)", c.ContainerId, c.Label)
	return utils.HandleDBError(err)
}

// NewContainerFromProto creates a new instance of Container populated from the given proto message.
func NewContainerFromProto(p *scheduler_proto.Container) *Container {
	return &Container{Container: *p}
}

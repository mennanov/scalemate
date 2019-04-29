package models

import (
	"fmt"
	"strings"
	"time"

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
	ListContainersMaxLimit = 300
)

// Container defines a Container model.
type Container struct {
	scheduler_proto.Container
	Node *Node
}

// ContainerLabel is a Container's label.
type ContainerLabel struct {
	ContainerId uint32
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

// ToProto populates the given `*scheduler_proto.Container` with the contents of the Container.
func (c *Container) ToProto(fieldMask *types.FieldMask) (*scheduler_proto.Container, error) {
	p := &c.Container

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask., generator.CamelCase)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		// Always include Container ID regardless of the mask.
		protoFiltered := &scheduler_proto.Container{Id: c.Id}
		if err := fieldmask_utils.StructToStruct(mask, p, protoFiltered); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return protoFiltered, nil
	}

	return p, nil
}

// Create inserts a new Container in DB and returns the corresponding event.
func (c *Container) Create(db utils.SqlxExtGetter) (*events_proto.Event, error) {
	if err := db.Get(c,
		`INSERT INTO containers (
	username, 
	status, 
	image, 
	cpu_class_min, cpu_class_max, 
	gpu_class_min, gpu_class_max, 
	disk_class_min, disk_class_max, 
	agent_auth_token) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *`,
		c.Username,
		fmt.Sprintf("%d", c.Status),
		c.Image,
		c.CpuClassMin, c.CpuClassMax,
		c.GpuClassMin, c.GpuClassMax,
		c.DiskClassMin, c.DiskClassMax,
		c.AgentAuthToken,
	); err != nil {
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
	event, err := events.NewEvent(containerProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	if err != nil {
		return nil, errors.Wrap(err, "events.NewEvent failed")
	}
	return event, nil
}

// LoadFromDB performs a lookup by ID and populates the struct.
func (c *Container) LoadFromDB(db utils.SqlxExtGetter, logger *logrus.Logger) error {
	if c.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't lookup Container with no ID"))
	}
	if err := utils.HandleDBError(db.Get(c, "SELECT * FROM containers WHERE id = $1", c.Id)); err != nil {
		return errors.Wrap(err, "failed to select Container from DB")
	}
	if err := c.LoadLabelsFromDB(db, logger); err != nil {
		return errors.Wrap(err, "container.LoadLabelsFromDB failed")
	}
	return nil
}

// LoadFromDBForUpdate is similar to LoadFromDB(), but locks the Container's row FOR UPDATE.
//func (c *Container) LoadFromDBForUpdate(db *gorm.DB, fields ...string) error {
//	return c.LoadFromDB(db.Set("gorm:query_option", "FOR UPDATE"), fields...)
//}

// LoadLabelsFromDB loads the corresponding ContainerLabels to the Labels field.
func (c *Container) LoadLabelsFromDB(db sqlx.Ext, logger *logrus.Logger) error {
	if c.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "Container is not saved in DB"))
	}
	rows, err := db.Queryx("SELECT label FROM container_labels WHERE container_id = $1", c.Id)
	if err != nil {
		return utils.HandleDBError(err)
	}
	defer utils.Close(rows, logger)

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
	db utils.SqlxNamedQuery,
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
				"status": fmt.Sprintf("%d", newStatus),
			}, logger)
		}
	}
	return nil, status.Errorf(codes.FailedPrecondition, "container with status %s can't be updated to %s",
		containerStatus.String(), newStatus.String())
}

// Update performs an UPDATE SqlxGetter query for the Container fields given in the `updates` argument and returns a
// corresponding event.
func (c *Container) Update(
	db utils.SqlxNamedQuery,
	updates map[string]interface{},
	logger *logrus.Logger,
) (*events_proto.Event, error) {
	if c.Id == 0 {
		return nil, errors.WithStack(status.Error(codes.FailedPrecondition, "can't update unsaved Container"))
	}
	keys := make([]string, len(updates))
	i := 0
	for k := range updates {
		keys[i] = fmt.Sprintf("%s=:%s", k, k)
		i++
	}
	rows, err := db.NamedQuery(fmt.Sprintf(
		"UPDATE containers SET %s WHERE id = %d RETURNING *", strings.Join(keys, ", "), c.Id), updates)
	if err != nil {
		return nil, utils.HandleDBError(err)
	}
	defer utils.Close(rows, logger)

	for rows.Next() {
		if err := rows.StructScan(c); err != nil {
			return nil, errors.Wrap(err, "rows.StructScan failed")
		}
	}

	fieldMask := &types.FieldMask{Paths: mapKeys(updates)}
	containerProto, err := c.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "job.ToProto failed")
	}
	event, err := events.NewEvent(containerProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER,
		fieldMask)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// CurrentLimit gets the most recently confirmed Limit for this Container.
func (c *Container) CurrentLimit(db *gorm.DB) (*Limit, error) {
	limit := &Limit{
		Container:   c,
		ContainerID: c.ID,
		Status:      utils.Enum(scheduler_proto.Limit_CONFIRMED),
	}
	if err := utils.HandleDBError(db.Where(limit).Order("id DESC").First(limit)); err != nil {
		return nil, errors.Wrap(err, "failed to find Limit")
	}
	return limit, nil
}

// NodesAvailable finds the best available Node to schedule this Container on.
func (c *Container) NodesAvailable(db *gorm.DB) ([]*Node, error) {
	if c.NodeID != nil {
		return nil, status.Error(codes.FailedPrecondition, "Container is already scheduled")
	}
	q := db.Set("gorm:query_option", "FOR UPDATE OF nodes").
		Table("nodes").
		Where("status = ?", utils.Enum(scheduler_proto.Node_ONLINE)).
		Where("cpu_class BETWEEN ? AND ?", c.CpuClassMin, c.CpuClassMax).
		Where("gpu_class BETWEEN ? AND ?", c.GpuClassMin, c.GpuClassMax).
		Where("disk_class BETWEEN ? AND ?", c.DiskClassMin, c.DiskClassMax).
		// FIXME: add labels constraints.
		Where("")

	var nodes []*Node
	if err := utils.HandleDBError(q.Find(&nodes)); err != nil {
		return nil, errors.Wrap(err, "failed to find Nodes")
	}

	return nodes, nil
}

// Schedule allocates resources on the given Node and sets the Container status to SCHEDULED.
func (c *Container) Schedule(db *gorm.DB, node *Node) ([]*events_proto.Event, error) {
	if c.ID == 0 {
		return nil, status.Error(codes.FailedPrecondition, "can't schedule unsaved Container")
	}
	if c.NodeID != nil {
		return nil, status.Error(codes.FailedPrecondition, "Container is already scheduled")
	}

	currentLimit, err := c.CurrentLimit(db)
	if err != nil {
		return nil, errors.Wrap(err, "container.CurrentLimit failed")
	}

	allocationUpdates, err := node.AllocateContainerLimit(currentLimit, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to allocate Node resources")
	}
	now := time.Now().UTC()
	allocationUpdates["scheduled_at"] = &now
	nodeUpdatedEvent, err := node.Update(db, allocationUpdates)
	if err != nil {
		return nil, errors.Wrap(err, "node.Update failed")
	}

	// Update the Container status to SCHEDULED.
	containerUpdatedEvent, err := c.UpdateStatus(db, scheduler_proto.Container_SCHEDULED)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update Container status")
	}

	return []*events_proto.Event{nodeUpdatedEvent, containerUpdatedEvent}, nil
}

// IsTerminated returns true if Container is in the final status (terminated).
func (c *Container) IsTerminated() bool {
	return c.Status == utils.Enum(scheduler_proto.Container_STOPPED) ||
		c.Status == utils.Enum(scheduler_proto.Container_FAILED) ||
		c.Status == utils.Enum(scheduler_proto.Container_CANCELLED) ||
		c.Status == utils.Enum(scheduler_proto.Container_DECLINED)
}

// ListContainers selects Containers from DB by the given filtering request.
// Returns the Containers slice, a total number of Containers that satisfy the criteria and an error.
func ListContainers(db *gorm.DB, request *scheduler_proto.ListContainersRequest) ([]Container, uint32, error) {
	query := db.Model(&Container{})
	ordering := request.GetOrdering()

	var orderBySQL string
	switch ordering {
	case scheduler_proto.ListContainersRequest_CREATED_AT_ASC:
		orderBySQL = "created_at"
	case scheduler_proto.ListContainersRequest_UPDATED_AT_ASC:
		orderBySQL = "updated_at"
	case scheduler_proto.ListContainersRequest_UPDATED_AT_DESC:
		orderBySQL = "updated_at DESC"
	default:
		orderBySQL = "created_at DESC"
	}
	query = query.Order(orderBySQL)

	// Filter by username.
	query = query.Where("username = ?", request.Username)

	// Filter by status.
	if len(request.Status) != 0 {
		enumStatus := make([]utils.Enum, len(request.Status))
		for i, s := range request.Status {
			enumStatus[i] = utils.Enum(s)
		}
		query = query.Where("status IN (?)", enumStatus)
	}

	// Perform a COUNT query with no limit and offset applied.
	var count uint32
	if err := utils.HandleDBError(query.Count(&count)); err != nil {
		return nil, 0, err
	}

	// Apply offset.
	query = query.Offset(request.GetOffset())

	// Apply limit.
	var limit uint32
	if request.GetLimit() <= ListContainersMaxLimit {
		limit = request.GetLimit()
	} else {
		limit = ListContainersMaxLimit
	}
	query = query.Limit(limit)

	// Perform a SELECT query.
	var containers []Container
	if err := utils.HandleDBError(query.Find(&containers)); err != nil {
		return nil, 0, err
	}
	return containers, count, nil
}

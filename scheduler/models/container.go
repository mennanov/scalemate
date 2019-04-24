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
	// ContainersScheduledForNodeQueryLimit is the number of Containers to be selected from DB when Containers are
	// selected to be scheduled on a Node.
	ContainersScheduledForNodeQueryLimit = 100
	// ListContainersMaxLimit is a maximum allowed limit in the SqlxGetter query that lists Containers.
	ListContainersMaxLimit = 300
)

// Container defines a Container gorm model.
type Container struct {
	utils.Model
	Node           *Node
	NodeID         *uint64    `gorm:"index" sql:"type:integer REFERENCES nodes(id)"`
	Username       string     `gorm:"type:varchar(32);index"`
	Status         utils.Enum `gorm:"type:smallint;not null"`
	StatusMessage  string
	Image          string
	CpuClassMin    utils.Enum `gorm:"type:smallint;index:idx_hdw_class"`
	CpuClassMax    utils.Enum `gorm:"type:smallint;index:idx_hdw_class"`
	GpuClassMin    utils.Enum `gorm:"type:smallint;index:idx_hdw_class"`
	GpuClassMax    utils.Enum `gorm:"type:smallint;index:idx_hdw_class"`
	DiskClassMin   utils.Enum `gorm:"type:smallint;index:idx_hdw_class"`
	DiskClassMax   utils.Enum `gorm:"type:smallint;index:idx_hdw_class"`
	AgentAuthToken []byte
	Labels         []*ContainerLabel
}

// ContainerLabel is a Container's label.
type ContainerLabel struct {
	ContainerID uint64 `gorm:"primary_key" sql:"type:integer REFERENCES containers(id)"`
	Label       string `gorm:"primary_key"`
}

// Create inserts a new ContainerLabel in DB.
func (c *ContainerLabel) Create(db *gorm.DB) error {
	return utils.HandleDBError(db.Create(c))
}

// NewContainerFromProto creates a new instance of Container populated from the given proto message.
func NewContainerFromProto(p *scheduler_proto.Container) (*Container, error) {
	container := &Container{
		Model: utils.Model{
			ID: p.Id,
		},
		Username:       p.Username,
		Status:         utils.Enum(p.Status),
		StatusMessage:  p.StatusMessage,
		Image:          p.Image,
		CpuClassMin:    utils.Enum(p.CpuClassMin),
		CpuClassMax:    utils.Enum(p.CpuClassMax),
		GpuClassMin:    utils.Enum(p.GpuClassMin),
		GpuClassMax:    utils.Enum(p.GpuClassMax),
		DiskClassMin:   utils.Enum(p.DiskClassMin),
		DiskClassMax:   utils.Enum(p.DiskClassMax),
		AgentAuthToken: p.AgentAuthToken,
	}

	if p.NodeId != 0 {
		container.NodeID = &p.NodeId
	}

	if p.CreatedAt != nil {
		createdAt, err := ptypes.Timestamp(p.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		container.CreatedAt = createdAt
	}

	if p.UpdatedAt != nil {
		updatedAt, err := ptypes.Timestamp(p.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		container.UpdatedAt = &updatedAt
	}

	for _, label := range p.Labels {
		container.Labels = append(container.Labels, &ContainerLabel{ContainerID: container.ID, Label: label})
	}

	return container, nil
}

// ToProto populates the given `*scheduler_proto.Container` with the contents of the Container.
func (c *Container) ToProto(fieldMask *field_mask.FieldMask) (*scheduler_proto.Container, error) {
	p := &scheduler_proto.Container{
		Id:             c.ID,
		Username:       c.Username,
		Status:         scheduler_proto.Container_Status(c.Status),
		StatusMessage:  c.StatusMessage,
		Image:          c.Image,
		CpuClassMin:    scheduler_proto.CPUClass(c.CpuClassMin),
		CpuClassMax:    scheduler_proto.CPUClass(c.CpuClassMax),
		GpuClassMin:    scheduler_proto.GPUClass(c.GpuClassMin),
		GpuClassMax:    scheduler_proto.GPUClass(c.GpuClassMax),
		DiskClassMin:   scheduler_proto.DiskClass(c.DiskClassMin),
		DiskClassMax:   scheduler_proto.DiskClass(c.DiskClassMax),
		AgentAuthToken: c.AgentAuthToken,
	}

	if c.NodeID != nil {
		p.NodeId = *c.NodeID
	}

	if !c.CreatedAt.IsZero() {
		createdAt, err := ptypes.TimestampProto(c.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.CreatedAt = createdAt
	}

	if c.UpdatedAt != nil {
		updatedAt, err := ptypes.TimestampProto(*c.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.UpdatedAt = updatedAt
	}

	for _, label := range c.Labels {
		p.Labels = append(p.Labels, label.Label)
	}

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask, generator.CamelCase)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		// Always include Container ID regardless of the mask.
		protoFiltered := &scheduler_proto.Container{Id: c.ID}
		if err := fieldmask_utils.StructToStruct(mask, p, protoFiltered); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return protoFiltered, nil
	}

	return p, nil
}

// Create inserts a new Container in DB and returns the corresponding event.
func (c *Container) Create(db *gorm.DB) (*events_proto.Event, error) {
	if err := utils.HandleDBError(db.Create(c)); err != nil {
		return nil, err
	}
	for _, label := range c.Labels {
		fmt.Println("label.Create for:", label)
		if err := label.Create(db); err != nil {
			return nil, errors.Wrap(err, "label.Create failed")
		}
	}
	containerProto, err := c.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "container.ToProto failed")
	}
	event, err := events.NewEvent(containerProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// LoadFromDB performs a lookup by ID and populates the struct.
func (c *Container) LoadFromDB(db *gorm.DB, fields ...string) error {
	if c.ID == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't lookup Container with no ID"))
	}
	if len(fields) > 0 {
		db = db.Select(fields)
	}
	if err := utils.HandleDBError(db.First(c, c.ID)); err != nil {
		return errors.Wrap(err, "failed to select Container from DB")
	}
	if err := c.LoadLabelsFromDB(db); err != nil {
		return errors.Wrap(err, "container.LoadLabelsFromDB failed")
	}
	return nil
}

// LoadFromDBForUpdate is similar to LoadFromDB(), but locks the Container's row FOR UPDATE.
func (c *Container) LoadFromDBForUpdate(db *gorm.DB, fields ...string) error {
	return c.LoadFromDB(db.Set("gorm:query_option", "FOR UPDATE"), fields...)
}

// LoadLabelsFromDB loads the corresponding ContainerLabels to the Labels field.
func (c *Container) LoadLabelsFromDB(db *gorm.DB) error {
	if c.ID == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "Container is not saved in DB"))
	}
	return utils.HandleDBError(db.Where("container_id = ?", c.ID).Find(&c.Labels))
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
func (c *Container) UpdateStatus(db *gorm.DB, newStatus scheduler_proto.Container_Status) (*events_proto.Event, error) {
	if c.ID == 0 {
		return nil, status.Error(codes.FailedPrecondition, "can't update status for an unsaved Container")
	}
	// Validate the containerStatus transition.
	containerStatus := scheduler_proto.Container_Status(c.Status)
	for _, s := range ContainerStatusTransitions[containerStatus] {
		if s == newStatus {
			now := time.Now().UTC()
			return c.Update(db, map[string]interface{}{
				"status":     utils.Enum(newStatus),
				"updated_at": &now,
			})
		}
	}
	return nil, status.Errorf(codes.FailedPrecondition, "container with status %s can't be updated to %s",
		containerStatus.String(), newStatus.String())
}

// Update performs an UPDATE SqlxGetter query for the Container fields given in the `updates` argument and returns a
// corresponding event.
func (c *Container) Update(db *gorm.DB, updates map[string]interface{}) (*events_proto.Event, error) {
	if c.ID == 0 {
		return nil, errors.WithStack(status.Error(codes.FailedPrecondition, "can't update not saved Container"))
	}
	if err := utils.HandleDBError(db.Model(c).Updates(updates)); err != nil {
		return nil, errors.Wrap(err, "failed to update a Container")
	}
	fieldMask := &field_mask.FieldMask{Paths: mapKeys(updates)}
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

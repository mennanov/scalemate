package models

import (
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/events/events_proto"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Node defines a Node model in DB.
// This model is a subset of the scheduler.models.Node model and holds a minimum information that is required to
// authenticate a Node instead of making a gRPC query to the Scheduler service.
// The data in this DB table is populated by listening to the Scheduler service events when a new Node is created.
type Node struct {
	gorm.Model
	Username string `gorm:"not null;unique_index:idx_node_username_name"`
	Name     string `gorm:"not null;unique_index:idx_node_username_name"`
	//revive:disable:var-naming
	CpuModel    string
	MemoryModel string
	GpuModel    string
	DiskModel   string
	//revive:enable:var-naming
}

// FromSchedulerProto populates the Node struct with values from `scheduler_proto.Node`.
func (node *Node) FromSchedulerProto(p *scheduler_proto.Node) {
	node.Username = p.Username
	node.Name = p.Name
	node.CpuModel = p.CpuModel
	node.MemoryModel = p.MemoryModel
	node.GpuModel = p.GpuModel
	node.DiskModel = p.DiskModel
}

// ToProto creates
func (node *Node) ToProto(fieldMask *field_mask.FieldMask) (*accounts_proto.Node, error) {
	createdAt, err := ptypes.TimestampProto(node.CreatedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	updatedAt, err := ptypes.TimestampProto(node.UpdatedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	p := &accounts_proto.Node{
		Id:          uint64(node.ID),
		Username:    node.Username,
		Name:        node.Name,
		CpuModel:    node.CpuModel,
		GpuModel:    node.GpuModel,
		MemoryModel: node.MemoryModel,
		DiskModel:   node.DiskModel,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		// Always include ID regardless of the field mask.
		pFiltered := &accounts_proto.Node{Id: uint64(node.ID)}
		if err := fieldmask_utils.StructToStruct(mask, p, pFiltered, generator.CamelCase, stringEye); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return pFiltered, nil
	}

	return p, nil
}

// Create creates a new Node in DB.
func (node *Node) Create(db *gorm.DB) (*events_proto.Event, error) {
	if node.ID != 0 {
		return nil, status.Error(codes.InvalidArgument, "can't create existing Node")
	}
	if err := utils.HandleDBError(db.Create(node)); err != nil {
		return nil, err
	}

	nodeProto, err := node.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "node.ToProto failed")
	}
	return events.NewEventFromPayload(nodeProto, events_proto.Event_CREATED, events_proto.Service_ACCOUNTS, nil)
}

// Get gets the Node from DB by a username and a Node name.
func (node *Node) Get(db *gorm.DB, username, name string) error {
	return utils.HandleDBError(db.Where("username = ? AND name = ?", username, name).First(node))
}

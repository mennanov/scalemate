package models

import (
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/utils"
)

// Limit is container's resources limits.
type Limit struct {
	utils.Model
	Container      *Container
	ContainerID    uint64 `gorm:"index" sql:"type:integer REFERENCES containers(id)"`
	Cpu            uint32
	Memory         uint32
	Gpu            uint32
	Disk           uint32
	NetworkIngress uint32
	NetworkEgress  uint32
	Status         utils.Enum `gorm:"type:smallint"`
	StatusMessage  string
	ConfirmedAt    *time.Time
}

// NewLimitFromProto returns a new instance of Limit populated from the given proto message.
func NewLimitFromProto(p *scheduler_proto.Limit) (*Limit, error) {
	limit := &Limit{
		Model: utils.Model{
			ID: p.Id,
		},
		Cpu:            p.Cpu,
		Memory:         p.Memory,
		Gpu:            p.Gpu,
		Disk:           p.Disk,
		NetworkIngress: p.NetworkIngress,
		NetworkEgress:  p.NetworkEgress,
		Status:         utils.Enum(p.Status),
		StatusMessage:  p.StatusMessage,
	}

	if p.CreatedAt != nil {
		createdAt, err := ptypes.Timestamp(p.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		limit.CreatedAt = createdAt
	}

	if p.UpdatedAt != nil {
		updatedAt, err := ptypes.Timestamp(p.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		limit.UpdatedAt = &updatedAt
	}

	if p.ConfirmedAt != nil {
		confirmedAt, err := ptypes.Timestamp(p.ConfirmedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		limit.ConfirmedAt = &confirmedAt
	}

	return limit, nil
}

// ToProto creates a scheduler_proto.Limit proto message.
func (l *Limit) ToProto(fieldMask *field_mask.FieldMask) (*scheduler_proto.Limit, error) {
	p := &scheduler_proto.Limit{
		Id:             l.ID,
		ContainerId:    l.ContainerID,
		Cpu:            l.Cpu,
		Memory:         l.Memory,
		Disk:           l.Disk,
		Gpu:            l.Gpu,
		NetworkIngress: l.NetworkIngress,
		NetworkEgress:  l.NetworkEgress,
		Status:         scheduler_proto.Limit_Status(l.Status),
		StatusMessage:  l.StatusMessage,
	}

	if !l.CreatedAt.IsZero() {
		createdAt, err := ptypes.TimestampProto(l.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.CreatedAt = createdAt
	}

	if l.UpdatedAt != nil {
		updatedAt, err := ptypes.TimestampProto(*l.UpdatedAt)
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
		// Always include Container ID regardless of the mask.
		protoFiltered := &scheduler_proto.Limit{Id: l.ID}
		if err := fieldmask_utils.StructToStruct(mask, p, protoFiltered); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return protoFiltered, nil
	}

	return p, nil
}


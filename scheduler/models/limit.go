package models

import (
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
)

// Limit is container's resources limits.
type Limit struct {
	scheduler_proto.Limit
}

// NewLimitFromProto returns a new instance of Limit populated from the given proto message.
func NewLimitFromProto(p *scheduler_proto.Limit) *Limit {
	return &Limit{Limit: *p}
}

// ToProto returns a proto Limit instance with applied proto field mask (if provided).
func (l *Limit) ToProto(fieldMask *types.FieldMask) (*scheduler_proto.Limit, error) {
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		return &l.Limit, nil
	}

	mask, err := fieldmask_utils.MaskFromPaths(fieldMask.Paths, generator.CamelCase)
	if err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
	}
	protoFiltered := &scheduler_proto.Limit{Id: l.Id}
	if err := fieldmask_utils.StructToStruct(mask, &l.Limit, protoFiltered); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
	}
	return protoFiltered, nil
}

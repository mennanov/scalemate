package models

import (
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// NodePricing represents a pricing policy for a Node.
type NodePricing struct {
	scheduler_proto.NodePricing
}

// ToProto returns a proto NodePricing instance with applied proto field mask (if provided).
func (n *NodePricing) ToProto(fieldMask *types.FieldMask) (*scheduler_proto.NodePricing, error) {
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		return &n.NodePricing, nil
	}

	mask, err := fieldmask_utils.MaskFromPaths(fieldMask.Paths, generator.CamelCase)
	if err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
	}
	protoFiltered := &scheduler_proto.NodePricing{Id: n.Id}
	if err := fieldmask_utils.StructToStruct(mask, &n.NodePricing, protoFiltered); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
	}
	return protoFiltered, nil
}

// Create inserts a new NodePricing in DB and returns the corresponding event.
func (n *NodePricing) Create(db utils.SqlxExtGetter) (*events_proto.Event, error) {
	data := map[string]interface{}{
		"node_id":      n.NodeId,
		"cpu_price":    n.CpuPrice,
		"memory_price": n.MemoryPrice,
		"gpu_price":    n.GpuPrice,
		"disk_price":   n.DiskPrice,
	}

	queryString, values, err := psq.Insert("node_pricing").SetMap(data).Suffix("RETURNING *").ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := db.Get(n, queryString, values...); err != nil {
		return nil, utils.HandleDBError(err)
	}

	nodePricing, err := n.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "NodePricing.ToProto failed")
	}
	return events.NewEvent(nodePricing, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil), nil
}

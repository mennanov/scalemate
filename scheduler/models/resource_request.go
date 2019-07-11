package models

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/utils"
)

// ResourceRequest is container's resources limit.
type ResourceRequest struct {
	scheduler_proto.ResourceRequest
}

// NewResourceRequestFromDB performs a lookup by ID and populates the struct.
func NewResourceRequestFromDB(db utils.SqlxExtGetter, requestId int64) (*ResourceRequest, error) {
	query := psq.Select("*").From("resource_requests").Where(sq.Eq{"id": requestId})

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

// NewResourceRequestFromDBVerifyNodeName similar to NewResourceRequestFromDB, but also checks if the provided Node name equals to the
// actual name of the corresponding Node in DB.
func NewResourceRequestFromDBVerifyNodeName(db utils.SqlxExtGetter, requestId int64, nodeName string) (*ResourceRequest, error) {
	query := psq.Select("r.*").From("resource_requests AS r").
		Join("containers AS c ON(r.container_id = c.id)").Join("nodes AS n ON(c.node_id = n.id)").
		Where("r.id = ? AND n.name = ?", requestId, nodeName)

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

// NewResourceRequestFromProto returns a new instance of ResourceRequest populated from the given proto message.
func NewResourceRequestFromProto(p *scheduler_proto.ResourceRequest) *ResourceRequest {
	return &ResourceRequest{ResourceRequest: *p}
}

// ToProto returns the corresponding scheduler_proto.Container instance with the mask applied.
func (r *ResourceRequest) ToProto(paths []string) (*scheduler_proto.ResourceRequest, error) {
	if len(paths) == 0 {
		return &r.ResourceRequest, nil
	}

	mask, err := fieldmask_utils.MaskFromPaths(paths, generator.CamelCase)
	if err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
	}
	// Always include ResourceRequest ID regardless of the mask.
	protoFiltered := &scheduler_proto.ResourceRequest{Id: r.Id}
	if err := fieldmask_utils.StructToStruct(mask, &r.ResourceRequest, protoFiltered); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
	}
	return protoFiltered, nil
}

// Create creates a new ResourceRequest in DB.
func (r *ResourceRequest) Create(db utils.SqlxExtGetter) error {
	data := map[string]interface{}{
		"container_id":   r.ContainerId,
		"cpu":            r.Cpu,
		"gpu":            r.Gpu,
		"memory":         r.Memory,
		"disk":           r.Disk,
		"status":         r.Status,
		"status_message": r.StatusMessage,
	}
	if r.Id != 0 {
		data["id"] = r.Id
	}
	queryString, args, err := psq.Insert("resource_requests").SetMap(data).Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	return utils.HandleDBError(db.Get(r, queryString, args...))
}

// ResourceRequestStatusTransitions defined possible Container status transitions.
var ResourceRequestStatusTransitions = map[scheduler_proto.ResourceRequest_Status][]scheduler_proto.ResourceRequest_Status{
	scheduler_proto.ResourceRequest_REQUESTED: {
		scheduler_proto.ResourceRequest_CONFIRMED,
		scheduler_proto.ResourceRequest_DECLINED,
	},
	scheduler_proto.ResourceRequest_CONFIRMED: {},
	scheduler_proto.ResourceRequest_DECLINED:  {},
}

// ValidateNewStatus checks whether the status of the ResourceRequest can be changed to the given one.
func (r *ResourceRequest) ValidateNewStatus(newStatus scheduler_proto.ResourceRequest_Status) error {
	// Validate the status transition.
	requestStatus := scheduler_proto.ResourceRequest_Status(r.Status)
	for _, s := range ResourceRequestStatusTransitions[requestStatus] {
		if s == newStatus {
			return nil
		}
	}
	return status.Errorf(codes.InvalidArgument, "resource request with status %s can't be updated to %s",
		requestStatus.String(), newStatus.String())
}

// Update performs an UPDATED query on the corresponding ResourceRequest.
func (r *ResourceRequest) Update(db utils.SqlxGetter, updates map[string]interface{}) error {
	if r.Id == 0 {
		return errors.WithStack(status.Error(codes.FailedPrecondition, "can't update unsaved Resource Request"))
	}
	query, args, err := psq.Update("resource_requests").SetMap(updates).Where(sq.Eq{"id": r.Id}).Suffix("RETURNING *").ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	return utils.HandleDBError(db.Get(r, query, args...))
}

// ToProtoFromUpdates returns a scheduler_proto.Container with the applied FieldMask derived from the given paths.
func (r *ResourceRequest) ToProtoFromUpdates(updates map[string]interface{}) (*scheduler_proto.ResourceRequest, *types.FieldMask, error) {
	paths := utils.MapKeys(updates)
	containerProto, err := r.ToProto(paths)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	fieldMask := &types.FieldMask{Paths: paths}
	return containerProto, fieldMask, nil
}

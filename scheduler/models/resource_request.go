package models

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

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

// NewResourceRequestFromDBLatest finds the most recently created Container's ResourceRequest.
func NewResourceRequestFromDBLatest(db utils.SqlxGetter, containerId int64) (*ResourceRequest, error) {
	query := psq.Select("*").From("resource_requests").
		Where("container_id = ? AND status = ?", containerId, scheduler_proto.ResourceRequest_CONFIRMED).
		Limit(1).
		OrderBy("id DESC")

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

// ToProto returns a proto ResourceRequest instance with applied proto field mask (if provided).
func (r *ResourceRequest) ToProto(fieldMask *types.FieldMask) (*scheduler_proto.ResourceRequest, error) {
	if fieldMask == nil || len(fieldMask.Paths) == 0 {
		return &r.ResourceRequest, nil
	}

	mask, err := fieldmask_utils.MaskFromPaths(fieldMask.Paths, generator.CamelCase)
	if err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
	}
	protoFiltered := &scheduler_proto.ResourceRequest{Id: r.Id}
	if err := fieldmask_utils.StructToStruct(mask, &r.ResourceRequest, protoFiltered); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
	}
	return protoFiltered, nil
}

// Create creates a new ResourceRequest in DB.
func (r *ResourceRequest) Create(db utils.SqlxExtGetter) (*events_proto.Event, error) {
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
		return nil, errors.WithStack(err)
	}

	if err := db.Get(r, queryString, args...); err != nil {
		return nil, utils.HandleDBError(err)
	}
	return &events_proto.Event{
		Payload: &events_proto.Event_SchedulerResourceRequestCreated{
			SchedulerResourceRequestCreated: &scheduler_proto.ResourceRequestCreated{
				ResourceRequest: &r.ResourceRequest,
			},
		},
		CreatedAt: time.Now().UTC(),
	}, nil
}

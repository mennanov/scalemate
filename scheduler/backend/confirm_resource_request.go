package backend

import (
	"context"
	"database/sql"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// ConfirmResourceRequest updates the status of the given ResourceRequest to CONFIRMED.
func (s *SchedulerBackend) ConfirmResourceRequest(ctx context.Context, lookupRequest *scheduler_proto.ResourceRequestLookupRequest) (*types.Empty, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get auth claims")
	}

	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}

	resourceRequest, err := models.NewResourceRequestFromDBVerifyNodeName(tx, lookupRequest.RequestId, claims.NodeName)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "resource request lookup failed"))
	}

	if err := resourceRequest.ValidateNewStatus(scheduler_proto.ResourceRequest_CONFIRMED); err != nil {
		return nil, errors.Wrap(err, "new status value validation failed")
	}

	updates := map[string]interface{}{"status": scheduler_proto.ResourceRequest_CONFIRMED}
	if err := resourceRequest.Update(tx, updates); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to update the resource request"))
	}

	resourceRequestProto, mask, err := resourceRequest.ToProtoFromUpdates(updates)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	if err := events.CommitAndPublish(tx, s.producer, &events_proto.Event{
		Payload: &events_proto.Event_SchedulerResourceRequestConfirmed{
			SchedulerResourceRequestConfirmed: &scheduler_proto.ResourceRequestConfirmed{
				ResourceRequest: resourceRequestProto,
				ResourceMask:    mask,
			},
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, errors.WithStack(err)
	}

	return &types.Empty{}, nil
}

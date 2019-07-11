package frontend

import (
	"context"
	"database/sql"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// AddResourceRequest adds a new ResourceRequest for a Container.
func (s *SchedulerFrontend) AddResourceRequest(
	ctx context.Context,
	request *scheduler_proto.AddResourceRequestRequest,
) (*scheduler_proto.ResourceRequest, error) {
	if err := ValidateResourceRequestFields(request.ResourceRequest, request.ResourceRequestMask); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := validation.ValidateStruct(request.ResourceRequest,
		validation.Field(&request.ResourceRequest.ContainerId, validation.Required),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}
	container, err := models.NewContainerFromDB(tx, request.ResourceRequest.ContainerId)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "could not get the corresponding container"))
	}
	if err := utils.ClaimsUsernameEqual(ctx, container.Username); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	if container.NodeId == nil {
		return nil, utils.RollbackTransaction(tx, status.Error(codes.FailedPrecondition, "container is not scheduled yet"))
	}
	if container.IsTerminated() {
		return nil, utils.RollbackTransaction(tx, status.Errorf(codes.FailedPrecondition, "container is terminated: %s", container.Status.String()))
	}

	existingRequests, err := container.ResourceRequests(tx)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to get container's resource requests"))
	}
	if len(existingRequests) == 0 {
		err = status.Error(codes.NotFound, "existing resource requests not found")
		// Normally this should never happen as a Container is always created with a ResourceRequest.
		s.logger.WithError(err).WithField("container", container).Error("unexpected error")
		return nil, utils.RollbackTransaction(tx, err)
	}

	var currentRequest *models.ResourceRequest
	for _, r := range existingRequests {
		if r.Status == scheduler_proto.ResourceRequest_REQUESTED {
			return nil, utils.RollbackTransaction(tx, status.Error(codes.FailedPrecondition, "resource request requested earlier has not been processed yet"))
		}
		if r.Status == scheduler_proto.ResourceRequest_CONFIRMED {
			currentRequest = r
		}
	}
	if currentRequest == nil {
		return nil, utils.RollbackTransaction(tx, status.Error(codes.FailedPrecondition, "no existing confirmed request found"))
	}

	resourceRequest := models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
		ContainerId: request.ResourceRequest.ContainerId,
		Cpu:         currentRequest.Cpu,
		Disk:        currentRequest.Disk,
		Memory:      currentRequest.Memory,
		Gpu:         currentRequest.Gpu,
	})
	if inMask("cpu", request.ResourceRequestMask) {
		resourceRequest.Cpu = request.ResourceRequest.Cpu
	}
	if inMask("gpu", request.ResourceRequestMask) {
		resourceRequest.Gpu = request.ResourceRequest.Gpu
	}
	if inMask("memory", request.ResourceRequestMask) {
		resourceRequest.Memory = request.ResourceRequest.Memory
	}
	if inMask("disk", request.ResourceRequestMask) {
		resourceRequest.Disk = request.ResourceRequest.Disk
	}

	node, err := models.NewNodeFromDB(tx, "id = ?", *container.NodeId)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	updates, err := node.AllocateResourcesUpdates(resourceRequest, currentRequest)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	if err := node.Update(tx, updates); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	if err := resourceRequest.Create(tx); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	if err := events.CommitAndPublish(tx, s.producer, &events_proto.Event{
		Payload: &events_proto.Event_SchedulerResourceRequestCreated{
			SchedulerResourceRequestCreated: &scheduler_proto.ResourceRequestCreated{
				ResourceRequest: &resourceRequest.ResourceRequest,
			},
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, errors.WithStack(err)
	}
	return &resourceRequest.ResourceRequest, nil
}

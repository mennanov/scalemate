package server

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *SchedulerServer) AddResourceRequest(
	ctx context.Context,
	request *scheduler_proto.AddResourceRequestRequest,
) (*scheduler_proto.ResourceRequest, error) {
	if err := ValidateResourceRequestFields(request.ResourceRequest, request.UpdatedFields); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := validation.ValidateStruct(request,
		validation.Field(&request.ContainerId, validation.Required),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	container, err := models.NewContainerFromDB(s.db, request.ContainerId)
	if err != nil {
		return nil, errors.Wrap(err, "could not get the corresponding container")
	}
	if err := utils.ClaimsUsernameEqual(ctx, container.Username); err != nil {
		return nil, errors.WithStack(err)
	}
	if container.NodeId == nil {
		return nil, status.Error(codes.FailedPrecondition, "container is not scheduled yet")
	}
	if container.IsTerminated() {
		return nil, status.Errorf(codes.FailedPrecondition, "container is %s", container.Status.String())
	}

	latestRequest, err := models.NewResourceRequestFromDBLatest(s.db, request.ContainerId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the latest existing resourceRequest")
	}
	if latestRequest.Status == scheduler_proto.ResourceRequest_REQUESTED {
		return nil, status.Error(codes.FailedPrecondition, "the resourceRequest requested earlier has not been processed yet")
	}

	resourceRequest := models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
		ContainerId: request.ContainerId,
		Cpu:         latestRequest.Cpu,
		Disk:        latestRequest.Disk,
		Memory:      latestRequest.Memory,
		Gpu:         latestRequest.Gpu,
	})
	if inMask("cpu", request.UpdatedFields) {
		resourceRequest.Cpu = request.ResourceRequest.Cpu
	}
	if inMask("gpu", request.UpdatedFields) {
		resourceRequest.Gpu = request.ResourceRequest.Gpu
	}
	if inMask("memory", request.UpdatedFields) {
		resourceRequest.Memory = request.ResourceRequest.Memory
	}
	if inMask("disk", request.UpdatedFields) {
		resourceRequest.Disk = request.ResourceRequest.Disk
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}
	node, err := models.NewNodeFromDB(tx, *container.NodeId)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	updates, err := node.AllocateResourcesUpdates(resourceRequest, latestRequest)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	err := node.Update(tx, updates)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	limitCreatedEvent, err := resourceRequest.Create(tx)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	if err := events.CommitAndPublish(tx, s.producer, nodeUpdatedEvent, limitCreatedEvent); err != nil {
		return nil, errors.WithStack(err)
	}
	return &resourceRequest.ResourceRequest, nil
}

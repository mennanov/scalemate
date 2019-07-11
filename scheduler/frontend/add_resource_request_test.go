package frontend_test

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestAddLimit() {
	initialRequest := &scheduler_proto.ResourceRequest{
		Cpu:    1,
		Memory: 1,
		Disk:   1,
		Gpu:    1,
	}
	stopHandlers := s.runEventHandlers()
	defer stopHandlers()

	// Create a Node a future Container will be scheduled on.
	_, err := s.backEndClient.CreateNode(context.Background(), &scheduler_proto.NodeWithPricing{
		Node: &scheduler_proto.Node{
			Name:            s.claimsInjector.Claims.NodeName,
			CpuAvailable:    2,
			MemoryAvailable: 2,
			GpuAvailable:    2,
			DiskAvailable:   2,
			Fingerprint:     []byte("fingerprint"),
		},
		NodePricing: &scheduler_proto.NodePricing{},
	})
	s.Require().NoError(err)

	// Connect the Node.
	connectNodeCtx, connectNodeCtxCanel := context.WithCancel(context.Background())
	defer connectNodeCtxCanel()
	connectClient, err := s.backEndClient.ConnectNode(connectNodeCtx, &scheduler_proto.ConnectNodeRequest{
		CpuAvailable:           2,
		MemoryAvailable:        2,
		DiskAvailable:          2,
		GpuAvailable:           2,
		NetworkIngressCapacity: 10,
		NetworkEgressCapacity:  10,
	})
	s.Require().NoError(err)
	go connectClient.Recv()

	// Create a Container future ResourceRequests will be added to.
	request := &scheduler_proto.ContainerWithResourceRequest{
		Container: &scheduler_proto.Container{
			Image:    "image",
		},
		ResourceRequest: initialRequest,
	}
	wait := testutils.ExpectMessages(s.sc, events.SchedulerSubjectName, s.logger, "Event_SchedulerContainerCreated")
	ctx := context.Background()
	container, err := s.frontEndClient.CreateContainer(ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(container)

	s.Require().NoError(wait(time.Second))

	// TODO: implement NodeConnected handler that will schedule this container as this test is flaky now: the Node may
	//  connect after the container is created.
	s.Run("failed precondition for unscheduled container", func() {
		response, err := s.frontEndClient.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ResourceRequest: &scheduler_proto.ResourceRequest{
				ContainerId: container.Container.Id,
				Cpu:         1,
				Memory:      1,
				Disk:        1,
				Gpu:         1,
			},
		})
		testutils.AssertErrorCode(s.T(), err, codes.FailedPrecondition)
		s.Nil(response)
	})

	wait = testutils.ExpectMessages(s.sc, events.SchedulerSubjectName, s.logger, "Event_SchedulerContainerScheduled")
	s.Require().NoError(wait(time.Second))

	// Confirm the corresponding initial ResourceRequest.
	_, err = s.backEndClient.ConfirmResourceRequest(context.Background(), &scheduler_proto.ResourceRequestLookupRequest{
		RequestId: container.ResourceRequest.Id,
	})
	s.Require().NoError(err)

	var resourceRequest *scheduler_proto.ResourceRequest
	s.Run("request added with non-empty mask", func() {
		wait := testutils.ExpectMessages(s.sc, events.SchedulerSubjectName, s.logger, "Event_SchedulerResourceRequestCreated")
		newCpuValue := uint32(2)
		resourceRequest, err = s.frontEndClient.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ResourceRequest: &scheduler_proto.ResourceRequest{
				ContainerId: container.Container.Id,
				Cpu:         newCpuValue,
				Gpu:         3,
				Disk:        3,
				Memory:      3,
			},
			ResourceRequestMask: &types.FieldMask{
				Paths: []string{"cpu"},
			},
		})
		s.Require().NoError(err)
		s.EqualValues(newCpuValue, resourceRequest.Cpu)
		s.Equal(initialRequest.Memory, resourceRequest.Memory)
		s.Equal(initialRequest.Gpu, resourceRequest.Gpu)
		s.Equal(initialRequest.Disk, resourceRequest.Disk)
		s.NoError(wait(time.Second))
	})

	s.Run("failed precondition for pending resource request", func() {
		response, err := s.frontEndClient.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ResourceRequest: &scheduler_proto.ResourceRequest{
				ContainerId: container.Container.Id,
				Cpu:         1,
				Memory:      1,
				Disk:        1,
				Gpu:         1,
			},
		})
		testutils.AssertErrorCode(s.T(), err, codes.FailedPrecondition)
		s.Nil(response)
	})

	s.Run("request added with empty mask", func() {
		// Confirm the recently created ResourceRequest.
		_, err = s.backEndClient.ConfirmResourceRequest(context.Background(), &scheduler_proto.ResourceRequestLookupRequest{
			RequestId: resourceRequest.Id,
		})
		s.Require().NoError(err)

		requested := &scheduler_proto.ResourceRequest{
			ContainerId: container.Container.Id,
			Cpu:         2,
			Memory:      2,
			Disk:        2,
			Gpu:         2,
		}
		wait := testutils.ExpectMessages(s.sc, events.SchedulerSubjectName, s.logger, "Event_SchedulerResourceRequestCreated")
		requestCreated, err := s.frontEndClient.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ResourceRequest: requested,
		})
		s.Require().NoError(err)
		s.Equal(requested.Cpu, requestCreated.Cpu)
		s.Equal(requested.Memory, requestCreated.Memory)
		s.Equal(requested.Gpu, requestCreated.Gpu)
		s.Equal(requested.Disk, requestCreated.Disk)
		s.Equal(scheduler_proto.ResourceRequest_REQUESTED, requestCreated.Status)
		s.NoError(wait(time.Second))
	})

	s.Run("permission denied", func() {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
			Username: "different_username",
		})
		defer restoreClaims()

		limit, err := s.frontEndClient.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ResourceRequest: &scheduler_proto.ResourceRequest{
				ContainerId: container.Container.Id,
				Cpu:         1,
				Gpu:         1,
				Disk:        1,
				Memory:      1,
			},
		})
		testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
		s.Nil(limit)
	})

	s.Run("not found for non existing container", func() {
		limit, err := s.frontEndClient.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ResourceRequest: &scheduler_proto.ResourceRequest{
				ContainerId: -1,
				Cpu:         1,
				Disk:        1,
				Memory:      1,
			},
		})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(limit)
	})
}

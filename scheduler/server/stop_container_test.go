package server_test

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestStopContainer() {
	createContainerRequest := &scheduler_proto.ContainerWithResourceRequest{
		Container: &scheduler_proto.Container{
			Username: s.claimsInjector.Claims.Username,
			Image:    "image",
		},
		ResourceRequest: &scheduler_proto.ResourceRequest{
			Cpu:    2,
			Memory: 256,
			Disk:   1024,
			Gpu:    1,
		},
	}
	containerWithResourceRequest, err := s.client.CreateContainer(context.Background(), createContainerRequest)
	s.Require().NoError(err)
	s.Require().NotNil(containerWithResourceRequest)

	ctx := context.Background()

	s.Run("updates the status", func() {
		_, err := s.client.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		s.Require().NoError(err)
		s.NoError(s.messagesHandler.ExpectMessages(
			testutils.KeyForEvent(&events_proto.Event{
				Type:    events_proto.Event_UPDATED,
				Service: events_proto.Service_SCHEDULER,
				Payload: &events_proto.Event_SchedulerContainer{
					SchedulerContainer: containerWithResourceRequest.Container,
				},
			}),
		))

		// Verify that the already STOPPED Container can't be stopped again.
		_, err = s.client.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		testutils.AssertErrorCode(s.T(), err, codes.FailedPrecondition)
	})

	s.Run("permission denied", func() {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
			Username: "different_username",
		})
		defer restoreClaims()

		container, err := s.client.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
		s.Nil(container)
	})

	s.Run("not found", func() {
		s.T().Parallel()
		_, err := s.client.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{ContainerId: 0})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
	})
}

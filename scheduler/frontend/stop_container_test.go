package frontend_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestStopContainer() {
	createContainerRequest := &scheduler_proto.ContainerWithResourceRequest{
		Container: &scheduler_proto.Container{
			Image:    "image",
		},
		ResourceRequest: &scheduler_proto.ResourceRequest{
			Cpu:    2,
			Memory: 256,
			Disk:   1024,
			Gpu:    1,
		},
	}
	containerWithResourceRequest, err := s.frontEndClient.CreateContainer(context.Background(), createContainerRequest)
	s.Require().NoError(err)
	s.Require().NotNil(containerWithResourceRequest)

	ctx := context.Background()

	s.Run("updates the status", func() {
		wait := testutils.ExpectMessages(s.sc, events.SchedulerSubjectName, s.logger, "Event_SchedulerContainerStopped")
		_, err := s.frontEndClient.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		s.Require().NoError(err)
		s.NoError(wait(time.Second))

		// Verify that the already STOPPED Container can't be stopped again.
		_, err = s.frontEndClient.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		testutils.AssertErrorCode(s.T(), err, codes.InvalidArgument)
	})

	s.Run("permission denied", func() {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
			Username: "different_username",
		})
		defer restoreClaims()

		container, err := s.frontEndClient.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
		s.Nil(container)
	})

	s.Run("not found", func() {
		s.T().Parallel()
		_, err := s.frontEndClient.StopContainer(ctx, &scheduler_proto.ContainerLookupRequest{ContainerId: 0})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
	})
}

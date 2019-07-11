package frontend_test

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestGetContainer() {
	request := &scheduler_proto.ContainerWithResourceRequest{
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
	containerWithResourceRequest, err := s.frontEndClient.CreateContainer(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(containerWithResourceRequest)

	ctx := context.Background()

	s.Run("get the created container", func() {
		s.T().Parallel()
		container, err := s.frontEndClient.GetContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		s.Require().NoError(err)
		s.Equal(containerWithResourceRequest.Container, container)
	})

	s.Run("permission denied", func() {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
			Username: "different_username",
		})
		defer restoreClaims()

		container, err := s.frontEndClient.GetContainer(ctx, &scheduler_proto.ContainerLookupRequest{
			ContainerId: containerWithResourceRequest.Container.Id})
		testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
		s.Nil(container)
	})

	s.Run("not found", func() {
		s.T().Parallel()
		container, err := s.frontEndClient.GetContainer(ctx, &scheduler_proto.ContainerLookupRequest{ContainerId: 0})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(container)
	})
}

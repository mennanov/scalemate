package frontend_test

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestListContainers() {
	validResourceRequest := &scheduler_proto.ResourceRequest{
		Cpu:    2,
		Memory: 256,
		Disk:   1024,
		Gpu:    1,
	}
	createContainerRequest := &scheduler_proto.ContainerWithResourceRequest{
		Container: &scheduler_proto.Container{
			Image: "image",
		},
		ResourceRequest: validResourceRequest,
	}
	ctx := context.Background()

	createdContainer, err := s.frontEndClient.CreateContainer(ctx, createContainerRequest)
	s.Require().NoError(err)
	s.Require().NotNil(createdContainer)

	s.Run("list the created container", func() {
		s.T().Parallel()
		response, err := s.frontEndClient.ListContainers(ctx, &scheduler_proto.ListContainersRequest{
			Limit: 10,
		})
		s.Require().NoError(err)
		s.EqualValues(1, response.TotalCount)
		s.Require().Equal(1, len(response.Containers))
		s.Equal(createdContainer.Container, response.Containers[0])
	})

	s.Run("not found", func() {
		s.T().Parallel()
		response, err := s.frontEndClient.ListContainers(ctx, &scheduler_proto.ListContainersRequest{
			Status: []scheduler_proto.Container_Status{scheduler_proto.Container_RUNNING},
			Limit:  10,
		})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(response)
	})

	s.Run("invalid argument", func() {
		s.T().Parallel()
		for _, request := range []*scheduler_proto.ListContainersRequest{
			{
			},
			{
				Limit: 0,
			},
		} {
			response, err := s.frontEndClient.ListContainers(ctx, request)
			testutils.AssertErrorCode(s.T(), err, codes.InvalidArgument)
			s.Nil(response)
		}
	})

	s.Run("not found for a different username", func() {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
			Username: "different_username",
		})
		defer restoreClaims()

		response, err := s.frontEndClient.ListContainers(ctx, &scheduler_proto.ListContainersRequest{
			Limit: 10,
		})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(response)
	})
}

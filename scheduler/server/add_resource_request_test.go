package server_test

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestAddLimit() {
	initialRequest := &scheduler_proto.ResourceRequest{
		Cpu:    2,
		Memory: 256,
		Disk:   1024,
		Gpu:    1,
	}
	validRequest := &scheduler_proto.ResourceRequest{
		Cpu:    4,
		Memory: 512,
		Disk:   2048,
		Gpu:    2,
	}

	// Create a Node a future Container will be scheduled on.
	node := &models.Node{
		Node: scheduler_proto.Node{
			CpuAvailable:    validRequest.Cpu - initialRequest.Cpu,
			MemoryAvailable: validRequest.Memory - initialRequest.Memory,
			GpuAvailable:    validRequest.Gpu - initialRequest.Gpu,
			DiskAvailable:   validRequest.Disk - initialRequest.Disk,
		},
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	request := &scheduler_proto.ContainerWithResourceRequest{
		Container: &scheduler_proto.Container{
			Username: s.claimsInjector.Claims.Username,
			Image:    "image",
		},
		ResourceRequest: initialRequest,
	}
	ctx := context.Background()
	containerWithLimit, err := s.client.CreateContainer(ctx, request)
	s.Require().NoError(err)
	s.Require().NotNil(containerWithLimit)

	s.Run("failed precondition for unscheduled container", func() {
		limitCreated, err := s.client.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ContainerId: containerWithLimit.Container.Id,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				Cpu: 5,
			},
			UpdatedFields: &types.FieldMask{
				Paths: []string{"cpu"},
			},
		})
		testutils.AssertErrorCode(s.T(), err, codes.FailedPrecondition)
		s.Nil(limitCreated)
	})

	// Manually update the status of the container to SCHEDULED.
	container := models.NewContainerFromProto(containerWithLimit.Container)
	_, err = container.Update(s.db, map[string]interface{}{
		"node_id": node.Id,
		"status":  scheduler_proto.Container_SCHEDULED,
	})
	s.Require().NoError(err)

	s.Run("request added with non-empty mask", func() {
		resourceRequestCreated, err := s.client.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ContainerId: containerWithLimit.Container.Id,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				Cpu: 4,
			},
			UpdatedFields: &types.FieldMask{
				Paths: []string{"cpu"},
			},
		})
		s.Require().NoError(err)
		s.EqualValues(4, resourceRequestCreated.Cpu)
		s.Equal(initialRequest.Memory, resourceRequestCreated.Memory)
		s.Equal(initialRequest.Gpu, resourceRequestCreated.Gpu)
		s.Equal(initialRequest.Disk, resourceRequestCreated.Disk)
	})

	//s.Run("limit added with empty mask", func() {
	//	s.T().Parallel()
	//	limitCreated, err := s.client.AddResourceRequest(ctx, &scheduler_proto.AddLimitsRequest{
	//		ContainerId: containerWithLimit.Container.Id,
	//		Limits:      validRequest,
	//	})
	//	s.Require().NoError(err)
	//	s.Equal(validRequest.Cpu, limitCreated.Cpu)
	//	s.Equal(validRequest.Memory, limitCreated.Memory)
	//	s.Equal(validRequest.Gpu, limitCreated.Gpu)
	//	s.Equal(validRequest.Disk, limitCreated.Disk)
	//	s.Equal(scheduler_proto.Limit_REQUESTED, limitCreated.Status)
	//})

	s.Run("permission denied", func() {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
			Username: "different_username",
		})
		defer restoreClaims()

		limit, err := s.client.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ContainerId:     containerWithLimit.Container.Id,
			ResourceRequest: validRequest,
		})
		testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
		s.Nil(limit)
	})

	s.Run("not found", func() {
		s.T().Parallel()
		limit, err := s.client.AddResourceRequest(ctx, &scheduler_proto.AddResourceRequestRequest{
			ContainerId:     containerWithLimit.Container.Id + 1,
			ResourceRequest: validRequest,
		})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(limit)
	})
}

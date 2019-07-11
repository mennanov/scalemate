package frontend_test

import (
	"context"
	"fmt"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestCreateContainer() {
	request := &scheduler_proto.ContainerWithResourceRequest{
		Container: &scheduler_proto.Container{
			Image:             "image",
			NetworkIngressMin: 50,
			NetworkEgressMin:  20,
			CpuClassMin:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
			CpuClassMax:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
			GpuClassMin:       scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
			GpuClassMax:       scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE,
			DiskClassMin:      scheduler_proto.DiskClass_DISK_CLASS_HDD,
			DiskClassMax:      scheduler_proto.DiskClass_DISK_CLASS_SSD,
			Labels:            []string{"label1", "label2"},
		},
		ResourceRequest: &scheduler_proto.ResourceRequest{
			Cpu:    2,
			Memory: 256,
			Disk:   1024,
			Gpu:    1,
		},
	}
	wait := testutils.ExpectMessages(s.sc, events.SchedulerSubjectName, s.logger, "Event_SchedulerContainerCreated")
	containerWithResourceRequest, err := s.frontEndClient.CreateContainer(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(containerWithResourceRequest)

	s.NoError(wait(time.Second))

	// Verify the Container was created.
	container, err := s.frontEndClient.GetContainer(context.Background(), &scheduler_proto.ContainerLookupRequest{
		ContainerId: containerWithResourceRequest.Container.Id})
	s.Require().NoError(err)
	s.Equal(containerWithResourceRequest.Container, container)
}

func (s *ServerTestSuite) TestCreateContainer_InvalidArgument() {
	ctx := context.Background()
	i64 := int64(42)
	validContainer := &scheduler_proto.Container{
		Username: s.claimsInjector.Claims.Username,
		Image:    "image",
	}
	validResourceRequest := &scheduler_proto.ResourceRequest{
		Cpu:    1,
		Memory: 64,
		Disk:   100,
	}
	now := time.Now()
	for i, request := range []*scheduler_proto.ContainerWithResourceRequest{
		{}, // Empty request.
		{
			Container:       validContainer,
			ResourceRequest: nil, // Invalid Request: can not be nil.
		},
		{
			Container:       nil, // Invalid Container: can not be nil.
			ResourceRequest: validResourceRequest,
		},
		// Requests with readonly fields filled in.
		{
			Container: &scheduler_proto.Container{
				Id:       1,
				Username: s.claimsInjector.Claims.Username,
				Image:    "image",
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: &scheduler_proto.Container{
				NodeId:   &i64,
				Username: s.claimsInjector.Claims.Username,
				Image:    "image",
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: &scheduler_proto.Container{
				Status:   scheduler_proto.Container_SCHEDULED,
				Username: s.claimsInjector.Claims.Username,
				Image:    "image",
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: &scheduler_proto.Container{
				StatusMessage: "message",
				Username:      s.claimsInjector.Claims.Username,
				Image:         "image",
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: &scheduler_proto.Container{
				CreatedAt: now,
				Username:  s.claimsInjector.Claims.Username,
				Image:     "image",
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: &scheduler_proto.Container{
				UpdatedAt: &now,
				Username:  s.claimsInjector.Claims.Username,
				Image:     "image",
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: &scheduler_proto.Container{
				NodeAuthToken: []byte("token"),
				Username:      s.claimsInjector.Claims.Username,
				Image:         "image",
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				Id: 1,
			},
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				ContainerId: 1,
			},
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				Status: scheduler_proto.ResourceRequest_CONFIRMED,
			},
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				StatusMessage: "status message",
			},
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				CreatedAt: now,
			},
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				UpdatedAt: &now,
			},
		},
		{
			Container: &scheduler_proto.Container{
				// Required "Image" field is missing.
				Username: s.claimsInjector.Claims.Username,
			},
			ResourceRequest: validResourceRequest,
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				// Required "Cpu" field is missing.
				Memory: 128,
				Disk:   1024,
			},
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				// Required "Memory" field is missing.
				Cpu:  1,
				Disk: 1024,
			},
		},
		{
			Container: validContainer,
			ResourceRequest: &scheduler_proto.ResourceRequest{
				// Required "Disk" field is missing.
				Cpu:    1,
				Memory: 128,
			},
		},
	} {
		// Capture loop variables.
		request := request
		i := i
		s.Run(fmt.Sprintf("invalid argument request-%d", i), func() {
			s.T().Parallel()
			response, err := s.frontEndClient.CreateContainer(ctx, request)
			testutils.AssertErrorCode(s.T(), err, codes.InvalidArgument)
			s.Nil(response)
		})
	}
}

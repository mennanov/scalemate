package server_test

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestReceiveTasks_BeforeJobCreated() {
	ctrl := gomock.NewController(s.T())
	accountsClient := NewMockAccountsClient(ctrl)
	defer ctrl.Finish()

	consumer, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	node := &models.Node{
		Username:        "node_owner",
		Name:            "node_name",
		Status:          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		CpuCapacity:     4,
		CpuAvailable:    2.5,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 8000,
		GpuCapacity:     4,
		GpuAvailable:    4,
		GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   20000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}

	_, err = node.Create(s.service.DB)
	s.Require().NoError(err)

	ctx := context.Background()

	accessToken := s.createToken(node.Username, node.Name, accounts_proto.User_USER, "access", time.Minute)
	jwtCredentials := auth.NewJWTCredentials(
		accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

	client, err := s.client.ReceiveTasks(ctx, &empty.Empty{}, grpc.PerRPCCredentials(jwtCredentials))
	s.Require().NoError(err)

	var taskFromNode *scheduler_proto.Task
	taskFromNodeReceived := make(chan struct{})
	go func(c chan struct{}) {
		taskFromNode, err = client.Recv()
		s.Require().NoError(err)
		c <- struct{}{}
	}(taskFromNodeReceived)

	jobRequest := &scheduler_proto.Job{
		Username:    "job_username",
		DockerImage: "postgres:11",
		CpuLimit:    1,
		CpuClass:    scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
		MemoryLimit: 4000,
		GpuLimit:    2,
		GpuClass:    scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
		DiskLimit:   10000,
		DiskClass:   scheduler_proto.DiskClass_DISK_CLASS_HDD,
	}
	jobAccessToken := s.createToken(jobRequest.Username, "", accounts_proto.User_USER, "access", time.Minute)

	jobJWTCredentials := auth.NewJWTCredentials(
		accountsClient, &accounts_proto.AuthTokens{AccessToken: jobAccessToken}, tokensFakeSaver)

	var taskFromJob *scheduler_proto.Task
	// Wait for the Node to be marked as ONLINE.
	for msg := range consumer {
		if strings.Contains(msg.RoutingKey, "scheduler.node.updated") {
			break
		}
	}
	// RunJob blocks until the Job is scheduled. It should happen immediately as there is a Node to schedule it for.
	taskFromJob, err = s.client.RunJob(ctx, jobRequest, grpc.PerRPCCredentials(jobJWTCredentials))
	s.Require().NoError(err)

	<-taskFromNodeReceived
	// By that time both client and node are expected to receive the same Task.
	s.Equal(taskFromJob.Id, taskFromNode.Id)
}

func (s *ServerTestSuite) TestReceiveTasks_AfterJobCreated() {
	ctrl := gomock.NewController(s.T())
	accountsClient := NewMockAccountsClient(ctrl)
	defer ctrl.Finish()

	// Create an online Node suitable for the Job, but with exhausted resources.
	nodeExhausted := &models.Node{
		Username:        "node_owner",
		Name:            "node_name1",
		Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    0,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 8000,
		GpuCapacity:     4,
		GpuAvailable:    0,
		GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   10000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := nodeExhausted.Create(s.service.DB)
	s.Require().NoError(err)

	node := &models.Node{
		Username:        "node_owner",
		Name:            "node_name2",
		Status:          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		CpuCapacity:     4,
		CpuAvailable:    2.5,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 8000,
		GpuCapacity:     4,
		GpuAvailable:    4,
		GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   20000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}

	_, err = node.Create(s.service.DB)
	s.Require().NoError(err)

	ctx := context.Background()

	// Create a Job before the Node is connected.
	jobRequest := &scheduler_proto.Job{
		Username:    "job_username",
		DockerImage: "postgres:11",
		CpuLimit:    1,
		CpuClass:    scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
		MemoryLimit: 4000,
		GpuLimit:    2,
		GpuClass:    scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
		DiskLimit:   10000,
		DiskClass:   scheduler_proto.DiskClass_DISK_CLASS_HDD,
	}
	jobAccessToken := s.createToken(jobRequest.Username, "", accounts_proto.User_USER, "access", time.Minute)

	jobJWTCredentials := auth.NewJWTCredentials(
		accountsClient, &accounts_proto.AuthTokens{AccessToken: jobAccessToken}, tokensFakeSaver)

	sg := &sync.WaitGroup{}
	var taskFromJob *scheduler_proto.Task
	go func(w *sync.WaitGroup) {
		// RunJob will block until the Job is scheduled, so it needs to be run in a background.
		taskFromJob, err = s.client.RunJob(ctx, jobRequest, grpc.PerRPCCredentials(jobJWTCredentials))
		s.Require().NoError(err)
		w.Done()
	}(sg)

	accessToken := s.createToken(node.Username, node.Name, accounts_proto.User_USER, "access", time.Minute)
	jwtCredentials := auth.NewJWTCredentials(
		accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

	client, err := s.client.ReceiveTasks(ctx, &empty.Empty{}, grpc.PerRPCCredentials(jwtCredentials))
	s.Require().NoError(err)

	sg.Add(2)
	var taskFromNode *scheduler_proto.Task
	go func(w *sync.WaitGroup) {
		taskFromNode, err = client.Recv()
		s.Require().NoError(err)
		w.Done()
	}(sg)

	sg.Wait()
	s.Equal(taskFromJob.Id, taskFromNode.Id)
}

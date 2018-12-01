package server_test

import (
	"context"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

var nodeOnline = &models.Node{
	Username:        "node_owner",
	Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
	CpuCapacity:     4,
	CpuAvailable:    2.5,
	CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
	CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
	MemoryCapacity:  8000,
	MemoryAvailable: 4000,
	GpuCapacity:     4,
	GpuAvailable:    2,
	GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
	GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
	DiskCapacity:    20000,
	DiskAvailable:   10000,
	DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	ConnectedAt:     time.Now(),
}

func (s *ServerTestSuite) TestRunJob() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	messages, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	_, err = nodeOnline.Create(s.service.DB)
	s.Require().NoError(err)
	// Reset the testing data structure for future tests.
	defer func() { nodeOnline.ID = 0 }()

	ctx := context.Background()
	req := &scheduler_proto.Job{
		Username:    "username",
		DockerImage: "postgres:latest",
		CpuLimit:    0.1,
		MemoryLimit: 128,
		DiskLimit:   1024,
	}
	accessToken := s.createToken("username", "", accounts_proto.User_USER, "access", time.Minute)
	accountsClient := NewMockAccountsClient(ctrl)
	jwtCredentials := auth.NewJWTCredentials(
		accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

	res, err := s.client.RunJob(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
	s.Require().NoError(err)
	s.NotNil(res)
	// Check that the Task is created and saved to DB.
	taskFromDB := &models.Task{}
	nodeFromDB := &models.Node{}
	jobFromDB := &models.Job{}
	s.Require().NoError(
		s.service.DB.First(taskFromDB).Related(nodeFromDB, "Node").Related(jobFromDB, "Job").Error)
	// Verify Task.
	s.Equal(nodeFromDB.ID, taskFromDB.NodeID)
	// Verify Node.
	s.Equal(nodeOnline.Username, nodeFromDB.Username)
	// Verify Job.
	s.Equal(req.Username, jobFromDB.Username)
	s.Equal(req.DockerImage, jobFromDB.DockerImage)
	s.Equal(req.CpuLimit, jobFromDB.CpuLimit)
	// Check that all messages have been sent.
	utils.WaitForMessages(messages, "scheduler.job.created")
}

func (s *ServerTestSuite) TestRunJob_Cancelled() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	messages, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	_, err = nodeOnline.Create(s.service.DB)
	s.Require().NoError(err)
	// Reset the testing data structure for future tests.
	defer func() { nodeOnline.ID = 0 }()

	// Create a context with a short timeout.
	ctx, cancel := context.WithCancel(context.Background())

	req := &scheduler_proto.Job{
		Username:    "username2",
		DockerImage: "postgres:latest",
		// CpuLimit of 99 is too high, so no Node is going to be found.
		CpuLimit:    99,
		MemoryLimit: 32,
		DiskLimit:   32,
	}
	accessToken := s.createToken("username", "", accounts_proto.User_USER, "access", time.Minute)
	accountsClient := NewMockAccountsClient(ctrl)
	jwtCredentials := auth.NewJWTCredentials(
		accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

	go func() {
		// Wait until the Job is created for this request.
		utils.WaitForMessages(messages, "scheduler.job.created")
		// Wait for the server to commit a transaction.
		time.Sleep(time.Millisecond * 100)
		// Cancel the ongoing RPC.
		cancel()
		// Wait till the cancelled context is handled on the service side.
		utils.WaitForMessages(messages, "scheduler.job.updated.status")
		// Wait for the server to commit a transaction.
		time.Sleep(time.Millisecond * 100)
		jobFromDB := &models.Job{}
		s.service.DB.Where("username = ?", req.Username).First(jobFromDB)
		// Verify Job is created and has a status CANCELLED.
		s.Equal(req.Username, jobFromDB.Username)
		s.Equal(req.DockerImage, jobFromDB.DockerImage)
		s.Equal(models.Enum(scheduler_proto.Job_STATUS_CANCELLED), jobFromDB.Status)
		s.Error(jobFromDB.LoadTasksFromDB(s.service.DB))
	}()
	res, err := s.client.RunJob(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
	s.assertGRPCError(err, codes.Canceled)

	s.Nil(res)
}

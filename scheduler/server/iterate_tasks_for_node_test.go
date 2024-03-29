package server_test

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestIterateTasksForNode_JobCreatedAfterNodeConnected() {
	node := &models.Node{
		Username:        "test_username",
		Name:            "node_name",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		CpuCapacity:     4,
		CpuAvailable:    2.5,
		CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 8000,
		GpuCapacity:     4,
		GpuAvailable:    4,
		GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   20000,
		DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}

	_, err := node.Create(s.db)
	s.Require().NoError(err)

	// Claims should contain a Node name.
	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
		Username: node.Username,
		NodeName: node.Name,
	})
	defer restoreClaims()

	ctx := context.Background()
	client, err := s.client.IterateTasksForNode(ctx, &empty.Empty{})
	s.Require().NoError(err)

	var taskForNode *scheduler_proto.Task
	taskReceivedByNode := make(chan struct{})
	go func(c chan struct{}) {
		taskForNode, err = client.Recv()
		s.Require().NoError(err)
		c <- struct{}{}
	}(taskReceivedByNode)

	// Wait for the Node to be marked ONLINE.
	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.node.updated`)

	jobRequest := &scheduler_proto.Job{
		Username:    "test_username",
		CpuLimit:    1,
		CpuClass:    scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
		MemoryLimit: 4000,
		GpuLimit:    2,
		GpuClass:    scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
		DiskLimit:   10000,
		DiskClass:   scheduler_proto.DiskClass_DISK_CLASS_HDD,
		RunConfig: &scheduler_proto.Job_RunConfig{
			Image: "image",
		},
	}
	jobProto, err := s.client.CreateJob(ctx, jobRequest)
	s.Require().NoError(err)
	// Manually update the Container's status to PENDING.
	job := &models.Container{}
	s.Require().NoError(job.FromProto(jobProto))
	jobUpdatedEvent, err := job.UpdateStatus(s.db, scheduler_proto.Job_STATUS_PENDING)
	s.Require().NoError(err)
	// Send the event about the new Container's status.
	s.Require().NoError(s.producer.Send(jobUpdatedEvent))
	events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.job.updated", "scheduler.task.created")
	<-taskReceivedByNode
	// Verify that the Task the Node has received is for the requested Container.
	s.Equal(jobProto.Id, taskForNode.JobId)
}

func (s *ServerTestSuite) TestIterateTasksForNode_NodeConnectedAfterJobCreated() {
	// Create an online Node suitable for the Container, but with exhausted resources.
	nodeOnlineExhausted := &models.Node{
		Username:        "test_username",
		Name:            "node_name1",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    0,
		CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 8000,
		GpuCapacity:     4,
		GpuAvailable:    0,
		GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   10000,
		DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := nodeOnlineExhausted.Create(s.db)
	s.Require().NoError(err)

	// Node that will connect afterwards.
	node := &models.Node{
		Username:        "test_username",
		Name:            "node_name2",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		CpuCapacity:     4,
		CpuAvailable:    2.5,
		CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 8000,
		GpuCapacity:     4,
		GpuAvailable:    4,
		GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   20000,
		DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err = node.Create(s.db)
	s.Require().NoError(err)

	ctx := context.Background()

	// Create a Container before the Node is connected.
	jobRequest := &scheduler_proto.Job{
		Username:    "test_username",
		CpuLimit:    1,
		CpuClass:    scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
		MemoryLimit: 4000,
		GpuLimit:    2,
		GpuClass:    scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
		DiskLimit:   10000,
		DiskClass:   scheduler_proto.DiskClass_DISK_CLASS_HDD,
		RunConfig: &scheduler_proto.Job_RunConfig{
			Image: "image",
		},
	}

	jobProto, err := s.client.CreateJob(ctx, jobRequest)
	s.Require().NoError(err)

	// Manually update the Container's status to PENDING.
	job := &models.Container{}
	s.Require().NoError(job.FromProto(jobProto))
	jobUpdatedEvent, err := job.UpdateStatus(s.db, scheduler_proto.Job_STATUS_PENDING)
	s.Require().NoError(err)
	// Send the event about the new Container's status.
	s.Require().NoError(s.producer.Send(jobUpdatedEvent))
	events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.job.updated")
	// Claims should contain a Node name.
	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
		Username: node.Username,
		NodeName: node.Name,
	})
	defer restoreClaims()

	client, err := s.client.IterateTasksForNode(ctx, &empty.Empty{})
	s.Require().NoError(err)
	taskForNode, err := client.Recv()
	s.Require().NoError(err)
	s.Equal(jobProto.Id, taskForNode.JobId)
}

func (s *ServerTestSuite) TestIterateTasksForNode_AlreadyConnected() {
	node := &models.Node{
		Username: s.claimsInjector.Claims.Username,
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	// Claims should contain a Node name.
	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
		Username: node.Username,
		NodeName: node.Name,
	})
	defer restoreClaims()

	// Connect the Node once.
	ctx, ctxCancel := context.WithCancel(context.Background())
	client1, err := s.client.IterateTasksForNode(ctx, &empty.Empty{})
	s.Require().NoError(err)

	errorsReceived := make(chan error)

	go func(e chan error) {
		task, err := client1.Recv()
		s.Require().Error(err)
		s.Require().Nil(task)
		e <- err
	}(errorsReceived)

	// Connect the Node twice.
	client2, err := s.client.IterateTasksForNode(ctx, &empty.Empty{})
	s.Require().NoError(err)

	go func(e chan error) {
		task, err := client2.Recv()
		s.Require().Error(err)
		s.Require().Nil(task)
		e <- err
	}(errorsReceived)

	// It's unclear which client will receive a response first, but the first error must be the one that is received by
	// the slowest client and this error is "FailedPrecondition" - the Node is already connected.
	s.assertGRPCError(<-errorsReceived, codes.FailedPrecondition)
	// Cancel the context which is shared by the both clients. It will terminate the in-progress request of the fastest
	// client which is still waiting for the Tasks to come.
	ctxCancel()
	s.assertGRPCError(<-errorsReceived, codes.Canceled)
	// Wait for the Node to be marked OFFLINE.
	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.node.updated\.((.*?disconnected_at.*?status)|(.*?status.*?disconnected_at))`)
}

func (s *ServerTestSuite) TestIterateTasksForNode_JobsUpdatedForDisconnectedNode() {
	node := &models.Node{
		Username:     s.claimsInjector.Claims.Username,
		Name:         "node_name",
		CpuAvailable: 1,
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Container{
		CpuLimit:      1,
		Status:        utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		RestartPolicy: utils.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	// Claims should contain a Node name.
	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
		Username: node.Username,
		NodeName: node.Name,
	})
	defer restoreClaims()

	ctx, ctxCancel := context.WithCancel(context.Background())
	client, err := s.client.IterateTasksForNode(ctx, &empty.Empty{})
	s.Require().NoError(err)

	task, err := client.Recv()
	s.Require().NoError(err)
	s.Equal(task.JobId, job.ID)

	// Wait for the Node to be marked ONLINE.
	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.node.updated\.((.*?connected_at.*?status)|(.*?status.*?connected_at))`)
	// Wait for the Task to be created.
	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.task.created`)

	// Load the corresponding Tasks from DB and verify their status.
	s.Require().NoError(job.LoadTasksFromDB(s.db))
	// Only 1 Task is expected.
	s.Require().Equal(1, len(job.Tasks))
	// Manually mark the Task as RUNNING.
	_, err = job.Tasks[0].UpdateStatus(s.db, scheduler_proto.Task_STATUS_RUNNING)
	s.Require().NoError(err)

	// Disconnect the Node.
	ctxCancel()
	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.node.updated\.((.*?disconnected_at.*?status)|(.*?status.*?disconnected_at))`)

	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.task.updated..*?status`)

	s.Require().NoError(job.Tasks[0].LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Task_STATUS_NODE_FAILED), job.Tasks[0].Status)

	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.job.updated..*?status`)
	// Verify that the Container's status is updated.
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Job_STATUS_FINISHED), job.Status)
}

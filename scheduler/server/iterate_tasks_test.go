package server_test

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestIterateTasks_IncludeExisting_TerminatedJob() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job1 := &models.Job{
		Username: "username_job",
		Status:   utils.Enum(scheduler_proto.Job_STATUS_FINISHED),
	}
	_, err = job1.Create(s.db)
	s.Require().NoError(err)

	job2 := &models.Job{
		Username: "username_job",
		Status:   utils.Enum(scheduler_proto.Job_STATUS_FINISHED),
	}
	_, err = job2.Create(s.db)
	s.Require().NoError(err)

	// Task that is expected to be returned.
	taskExistingJob1 := &models.Task{
		NodeID: node.ID,
		JobID:  job1.ID,
	}
	_, err = taskExistingJob1.Create(s.db)
	s.Require().NoError(err)

	// This Task should not be returned as it's for a different Job.
	taskExistingJob2 := &models.Task{
		NodeID: node.ID,
		JobID:  job2.ID,
	}
	_, err = taskExistingJob2.Create(s.db)
	s.Require().NoError(err)

	ctx := context.Background()
	req := &scheduler_proto.IterateTasksRequest{
		JobId:           job1.ID,
		IncludeExisting: true,
	}

	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: job1.Username})
	defer restoreClaims()

	client, err := s.client.IterateTasks(ctx, req)
	s.Require().NoError(err)
	var receivedTasks []*scheduler_proto.Task
	for {
		taskProto, err := client.Recv()
		if err == io.EOF {
			break
		}
		s.Require().NoError(err)
		receivedTasks = append(receivedTasks, taskProto)
	}
	// Verify that only the Task taskExistingJob1 is received.
	s.Require().Len(receivedTasks, 1)
	s.Equal(taskExistingJob1.ID, receivedTasks[0].Id)
}

func (s *ServerTestSuite) TestIterateTasks_IncludeExisting_NewTaskIsCreatedWhileIterating() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job1 := &models.Job{
		Username:      "username_job",
		Status:        utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: utils.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job1.Create(s.db)
	s.Require().NoError(err)

	job2 := &models.Job{
		Username: "username_job",
		Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
	}
	_, err = job2.Create(s.db)
	s.Require().NoError(err)

	taskExistingJob1 := &models.Task{
		NodeID: node.ID,
		JobID:  job1.ID,
	}
	_, err = taskExistingJob1.Create(s.db)
	s.Require().NoError(err)

	taskExistingJob2 := &models.Task{
		NodeID: node.ID,
		JobID:  job2.ID,
	}
	_, err = taskExistingJob2.Create(s.db)
	s.Require().NoError(err)

	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: job1.Username})
	defer restoreClaims()

	ctx := context.Background()
	req := &scheduler_proto.IterateTasksRequest{
		JobId:           job1.ID,
		IncludeExisting: true,
	}
	client, err := s.client.IterateTasks(ctx, req)
	s.Require().NoError(err)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func(wg *sync.WaitGroup) {
		// Give some time to the other go routine to receive existing Tasks.
		time.Sleep(time.Millisecond * 100)

		taskNewJob1 := &models.Task{
			JobID:  job1.ID,
			NodeID: node.ID,
			Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
		}
		taskCreatedEvent, err := taskNewJob1.Create(s.db)
		s.Require().NoError(err)

		s.Require().NoError(s.producer.Send(taskCreatedEvent))
		// Wait for the message to be received.
		events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.task.created")
		// Terminate the Task. The corresponding Job will be marked as finished and the Tasks channel will be closed.
		taskUpdatedEvent, err := taskNewJob1.UpdateStatus(s.db, scheduler_proto.Task_STATUS_FINISHED)
		s.Require().NoError(err)
		s.Require().NoError(s.producer.Send(taskUpdatedEvent))
		events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.task.updated", "scheduler.job.updated")
		wg.Done()
	}(wg)

	go func(wg *sync.WaitGroup) {
		var receivedTasks []*scheduler_proto.Task
		for {
			taskProto, err := client.Recv()
			if err == io.EOF {
				break
			}
			s.Require().NoError(err)
			receivedTasks = append(receivedTasks, taskProto)
		}
		s.Require().Len(receivedTasks, 2)
		s.Equal(taskExistingJob1.ID, receivedTasks[0].Id)
		s.NotEqual(taskExistingJob1.ID, receivedTasks[1].Id)
		wg.Done()
	}(wg)

	wg.Wait()
}

func (s *ServerTestSuite) TestIterateTasks_NotIncludeExisting_NewTaskIsCreatedWhileIterating() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job1 := &models.Job{
		Username:      "username_job",
		Status:        utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: utils.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job1.Create(s.db)
	s.Require().NoError(err)

	job2 := &models.Job{
		Username: "username_job",
		Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
	}
	_, err = job2.Create(s.db)
	s.Require().NoError(err)

	// Existing Task that is not expected to be returned.
	taskExistingJob1 := &models.Task{
		NodeID: node.ID,
		JobID:  job1.ID,
	}
	_, err = taskExistingJob1.Create(s.db)
	s.Require().NoError(err)

	// Another existing Task for a different Job that is not expected to be returned.
	taskExistingJob2 := &models.Task{
		NodeID: node.ID,
		JobID:  job2.ID,
	}
	_, err = taskExistingJob2.Create(s.db)
	s.Require().NoError(err)

	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: job1.Username})
	defer restoreClaims()

	ctx := context.Background()
	req := &scheduler_proto.IterateTasksRequest{
		JobId:           job1.ID,
		IncludeExisting: false,
	}
	client, err := s.client.IterateTasks(ctx, req)
	s.Require().NoError(err)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	var newTaskId uint64

	go func(wg *sync.WaitGroup) {
		taskNewJob1 := &models.Task{
			JobID:  job1.ID,
			NodeID: node.ID,
			Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
		}
		taskCreatedEvent, err := taskNewJob1.Create(s.db)
		s.Require().NoError(err)
		newTaskId = taskNewJob1.ID

		s.Require().NoError(s.producer.Send(taskCreatedEvent))
		// Wait for the message to be received.
		events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.task.created")
		// Give the service some time to process this event.
		// Terminate the Task. The corresponding Job will be marked as finished and the Tasks channel will be closed.
		taskUpdatedEvent, err := taskNewJob1.UpdateStatus(s.db, scheduler_proto.Task_STATUS_FINISHED)
		s.Require().NoError(err)
		s.Require().NoError(s.producer.Send(taskUpdatedEvent))
		events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.job.updated")
		wg.Done()
	}(wg)

	go func(wg *sync.WaitGroup) {
		var receivedTasks []*scheduler_proto.Task
		for {
			taskProto, err := client.Recv()
			if err == io.EOF {
				break
			}
			s.Require().NoError(err)
			receivedTasks = append(receivedTasks, taskProto)
		}
		s.Require().Len(receivedTasks, 1)
		s.Equal(newTaskId, receivedTasks[0].Id)
		wg.Done()
	}(wg)

	wg.Wait()
}

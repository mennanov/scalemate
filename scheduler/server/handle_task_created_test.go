package server_test

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestHandleTaskCreated_SendsTaskToAppropriateChannels() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{
		// Putting the same username as for the Node above to simplify claims injection.
		Username: node.Username,
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		NodeID: node.ID,
		JobID:  job.ID,
	}
	taskCreatedEvent, err := task.Create(s.db)
	s.Require().NoError(err)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	ctx := context.Background()

	// Claims should contain a Node name.
	restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{
		Username: node.Username,
		NodeName: node.Name,
	})
	defer restoreClaims()

	iterateTasksClient, err := s.client.IterateTasks(ctx, &scheduler_proto.IterateTasksRequest{JobId: job.ID})
	s.Require().NoError(err)
	go func(wg *sync.WaitGroup, task *models.Task) {
		defer wg.Done()

		receivedTask, err := iterateTasksClient.Recv()
		s.Require().NoError(err)
		s.Equal(receivedTask.Id, task.ID)
	}(wg, task)

	iterateTasksForNodeClient, err := s.client.IterateTasksForNode(ctx, &empty.Empty{})
	s.Require().NoError(err)
	go func(wg *sync.WaitGroup, task *models.Task) {
		defer wg.Done()

		receivedTask, err := iterateTasksForNodeClient.Recv()
		s.Require().NoError(err)
		s.Equal(receivedTask.Id, task.ID)
	}(wg, task)

	// Wait for the Node to be marked ONLINE.
	utils.WaitForMessages(s.amqpRawConsumer, `scheduler.node.updated`)

	s.Require().NoError(s.service.HandleTaskCreated(taskCreatedEvent))
	wg.Wait()
}

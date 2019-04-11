package server_test

import (
	"context"
	"io"
	"sync"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestHandleJobTerminated_ClientStopsReceivingTasks() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{
		Username: s.claimsInjector.Claims.Username,
		Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	jobUpdatedEvent, err := job.UpdateStatus(s.db, scheduler_proto.Job_STATUS_FINISHED)
	s.Require().NoError(err)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	iterateTasksClient, err := s.client.IterateTasks(
		context.Background(), &scheduler_proto.IterateTasksRequest{JobId: job.ID})
	s.Require().NoError(err)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		receivedTask, err := iterateTasksClient.Recv()
		s.Equal(io.EOF, err)
		s.Nil(receivedTask)
	}(wg)

	s.Require().NoError(s.service.HandleJobTerminated(jobUpdatedEvent))
	wg.Wait()
}

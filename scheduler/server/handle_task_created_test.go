package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestHandleTaskCreated_SendsTaskToAppropriateChannels() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	tasksForNode := make(chan *scheduler_proto.Task)
	s.service.NewTasksByNodeIDMutex.Lock()
	s.service.NewTasksByNodeID[node.ID] = tasksForNode
	s.service.NewTasksByNodeIDMutex.Unlock()

	tasksByJob := make(chan *scheduler_proto.Task)
	s.service.NewTasksByJobIDMutex.Lock()
	s.service.NewTasksByJobID[job.ID] = tasksByJob
	s.service.NewTasksByJobIDMutex.Unlock()

	task := &models.Task{
		NodeID: node.ID,
		JobID:  job.ID,
	}
	taskCreatedEvent, err := task.Create(s.db)
	s.Require().NoError(err)

	tasksReceived := make(chan struct{})
	go func() {
		// This Task is expected to be sent to the appropriate channels.
		taskForNode := <-tasksForNode
		taskByJob := <-tasksByJob

		s.Equal(taskByJob.Id, taskForNode.Id)
		tasksReceived <- struct{}{}
	}()

	s.Require().NoError(s.service.HandleTaskCreated(taskCreatedEvent))
	<-tasksReceived
}

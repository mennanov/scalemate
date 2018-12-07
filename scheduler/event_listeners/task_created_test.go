package event_listeners_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/event_listeners"
)

func (s *EventListenersTestSuite) TestTaskCreatedHandler_SendsTaskToAppropriateChannels() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	job := &models.Job{}
	_, err = job.Create(s.service.DB)
	s.Require().NoError(err)

	tasksForNode := make(chan *scheduler_proto.Task)
	s.service.NewTasksByNodeID[node.ID] = tasksForNode

	tasksByJob := make(chan *scheduler_proto.Task)
	s.service.NewTasksByJobID[job.ID] = tasksByJob

	task := &models.Task{
		NodeID: node.ID,
		JobID:  job.ID,
	}
	taskCreatedEvent, err := task.Create(s.service.DB)
	s.Require().NoError(err)

	tasksReceived := make(chan struct{})
	go func() {
		// This Task is expected to be sent to the appropriate channels.
		taskForNode := <-tasksForNode
		taskByJob := <-tasksByJob

		s.Equal(taskByJob.Id, taskForNode.Id)
		tasksReceived <- struct{}{}
	}()

	s.Require().NoError(event_listeners.TaskCreatedAMQPEventListener.Handler(s.service, taskCreatedEvent))
	<-tasksReceived
}

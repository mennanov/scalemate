package event_listeners_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/event_listeners"
	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *EventListenersTestSuite) TestJobTerminatedHandler_CorrespondingChannelIsClosed() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	job := &models.Job{}
	_, err = job.Create(s.service.DB)
	s.Require().NoError(err)

	jobUpdatedEvent, err := job.UpdateStatus(s.service.DB, scheduler_proto.Job_STATUS_FINISHED)
	s.Require().NoError(err)

	tasksByJob := make(chan *scheduler_proto.Task)
	s.service.NewTasksByJobID[job.ID] = tasksByJob

	channelClosed := make(chan struct{})
	go func() {
		_, ok := <-tasksByJob
		s.False(ok)
		channelClosed <- struct{}{}
	}()

	s.Require().NoError(event_listeners.JobTerminatedAMQPEventListener.Handler(s.service, jobUpdatedEvent))
	<-channelClosed
}

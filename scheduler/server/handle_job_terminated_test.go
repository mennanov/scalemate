package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestHandleJobTerminated_CorrespondingChannelIsClosed() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{Status: models.Enum(scheduler_proto.Job_STATUS_SCHEDULED)}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	jobUpdatedEvent, err := job.UpdateStatus(s.db, scheduler_proto.Job_STATUS_FINISHED)
	s.Require().NoError(err)

	tasksByJob := make(chan *scheduler_proto.Task)
	s.service.NewTasksByJobID[job.ID] = tasksByJob

	channelClosed := make(chan struct{})
	go func() {
		_, ok := <-tasksByJob
		s.False(ok)
		channelClosed <- struct{}{}
	}()

	s.Require().NoError(s.service.HandleJobTerminated(jobUpdatedEvent))
	<-channelClosed
}

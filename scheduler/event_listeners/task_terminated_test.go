package event_listeners_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/event_listeners"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *EventListenersTestSuite) TestTaskTerminatedHandler_CorrespondingJobIsTerminated() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	job := &models.Job{
		Username:      "job1",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job.Create(s.service.DB)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.service.DB)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.service.DB, scheduler_proto.Task_STATUS_FINISHED)
	s.Require().NoError(err)

	s.Require().NoError(event_listeners.TaskTerminatedAMQPEventListener.Handler(s.service, taskUpdatedEvent))

	s.Len(s.service.Publisher.(*events.FakeProducer).SentEvents, 1)

	// Verify that the jobScheduled now has a status "FINISHED".
	s.Require().NoError(job.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_FINISHED), job.Status)
}

func (s *EventListenersTestSuite) TestTaskTerminatedHandler_CorrespondingJobIsPending() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	job := &models.Job{
		Username:      "job1",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_FAILURE),
	}
	_, err = job.Create(s.service.DB)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.service.DB)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.service.DB, scheduler_proto.Task_STATUS_FAILED)
	s.Require().NoError(err)

	s.Require().NoError(event_listeners.TaskTerminatedAMQPEventListener.Handler(s.service, taskUpdatedEvent))

	s.Len(s.service.Publisher.(*events.FakeProducer).SentEvents, 1)

	// Verify that the jobScheduled now has a status "PENDING".
	s.Require().NoError(job.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_PENDING), job.Status)
}

package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestTaskTerminatedHandler_CorrespondingJobIsTerminated() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{
		Username:      "job1",
		Status:        models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.db, scheduler_proto.Task_STATUS_FINISHED)
	s.Require().NoError(err)

	s.Require().NoError(s.service.HandleTaskTerminated(taskUpdatedEvent))

	utils.WaitForMessages(s.amqpRawConsumer, `scheduler.job.updated\..*?status`)

	// Verify that the jobScheduled now has a status "FINISHED".
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_FINISHED), job.Status)
}

func (s *ServerTestSuite) TestTaskTerminatedHandler_CorrespondingJobIsPending() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{
		Username:      "job1",
		Status:        models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_FAILURE),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.db, scheduler_proto.Task_STATUS_FAILED)
	s.Require().NoError(err)

	s.Require().NoError(s.service.HandleTaskTerminated(taskUpdatedEvent))

	utils.WaitForMessages(s.amqpRawConsumer, `scheduler.job.updated\..*?status`)

	// Verify that the jobScheduled now has a status "PENDING".
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_PENDING), job.Status)
}

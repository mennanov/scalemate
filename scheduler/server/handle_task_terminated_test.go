package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
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
		Status:        utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: utils.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.db, scheduler_proto.Task_STATUS_FINISHED)
	s.Require().NoError(err)

	s.Require().NoError(s.service.HandleTaskTerminated(taskUpdatedEvent))

	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.job.updated\..*?status`)
	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.node.updated\..*?tasks_finished`)

	// Verify that the jobScheduled now has a status "FINISHED".
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Job_STATUS_FINISHED), job.Status)
	// Verify than the Node's statistics are updated.
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(uint64(1), node.TasksFinished)
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
		Status:        utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: utils.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_FAILURE),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.db, scheduler_proto.Task_STATUS_NODE_FAILED)
	s.Require().NoError(err)

	s.Require().NoError(s.service.HandleTaskTerminated(taskUpdatedEvent))

	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.job.updated\..*?status`)
	events.WaitForMessages(s.amqpRawConsumer, nil, `scheduler.node.updated\..*?tasks_failed`)

	// Verify that the jobScheduled now has a status "PENDING".
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Job_STATUS_PENDING), job.Status)
	// Verify than the Node's statistics are updated.
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(uint64(1), node.TasksFailed)
}

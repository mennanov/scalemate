package server_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestHandleNodeDisconnected_UpdatesCorrespondingTasks() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{}
	_, err = job.Create(s.db)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)

	now := time.Now()
	nodeUpdatedEvent, err := node.Updates(s.db, map[string]interface{}{
		"disconnected_at": &now,
		"status":          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
	})
	s.Require().NoError(err)

	s.Require().NoError(s.service.HandleNodeDisconnected(nodeUpdatedEvent))
	// Verify that the Task's status is updated.
	s.Require().NoError(task.LoadFromDB(s.db))
}

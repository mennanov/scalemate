package event_listeners_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/event_listeners"
	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *EventListenersTestSuite) TestNodeDisconnectedHandler_UpdatesCorrespondingTasks() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	job := &models.Job{}
	_, err = job.Create(s.service.DB)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.service.DB)

	nodeUpdatedEvent, err := node.Updates(s.service.DB, map[string]interface{}{
		"disconnected_at": time.Now(),
		"status":          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
	})
	s.Require().NoError(err)

	s.Require().NoError(event_listeners.NodeDisconnectedAMQPEventListener.Handler(s.service, nodeUpdatedEvent))
	// Verify that the Task's status is updated.
	s.Require().NoError(task.LoadFromDB(s.service.DB))
}

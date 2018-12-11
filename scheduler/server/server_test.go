package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestConnectNode() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	// Connect the Node to the service.
	connectEvent, err := s.service.ConnectNode(s.service.DB, node)
	s.Require().NoError(err)
	s.NotNil(connectEvent)
	// Verify that the Node is Online now.
	s.Require().NoError(node.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Node_STATUS_ONLINE), node.Status)
	// Check that there is a Tasks channel for this Node.
	_, ok := s.service.NewTasksByNodeID[node.ID]
	s.True(ok)
}

func (s *ServerTestSuite) TestDisconnectNode_ConnectThenDisconnect_UpdatesNodeJobTaskStatuses() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	// Connect the Node to the service.
	connectEvent, err := s.service.ConnectNode(s.service.DB, node)
	s.Require().NoError(err)
	s.NotNil(connectEvent)

	job := &models.Job{
		Status:        models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job.Create(s.service.DB)
	s.Require().NoError(err)

	task := &models.Task{
		NodeID: node.ID,
		JobID:  job.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.service.DB)
	s.Require().NoError(err)

	consumer, err := events.NewAMQPRawConsumer(s.amqpChannel, events.SchedulerAMQPExchangeName, "", "#")
	s.Require().NoError(err)

	// Disconnect the node.
	nodeDisconnectedEvent, err := s.service.DisconnectNode(s.service.DB, node)
	s.Require().NoError(err)
	s.Require().NoError(s.service.Publisher.Send(nodeDisconnectedEvent))
	utils.WaitForMessages(consumer, "scheduler.node.updated", "scheduler.task.updated", "scheduler.job.updated")
	// Verify that the corresponding Tasks and Jobs statuses are updated.
	s.Require().NoError(task.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Task_STATUS_NODE_FAILED), task.Status)
	s.Require().NoError(task.LoadJobFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_FINISHED), task.Job.Status)

	// Check that there is no Tasks channel for this Node anymore.
	_, ok := s.service.NewTasksByNodeID[node.ID]
	s.False(ok)
}

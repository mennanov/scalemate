package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestConnectNode() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	// Connect the Node to the service.
	connectEvent, err := s.service.ConnectNode(s.db, node)
	s.Require().NoError(err)
	s.NotNil(connectEvent)
	// Verify that the Node is marked as Online in DB now.
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Node_STATUS_ONLINE), node.Status)
	// Verify that the consecutive call to ConnectNode fails.
	_, err = s.service.ConnectNode(s.db, node)
	s.Equal(server.ErrNodeAlreadyConnected, err)
}

func (s *ServerTestSuite) TestDisconnectNode_ConnectThenDisconnect_UpdatesNodeJobTaskStatuses() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	// Connect the Node to the service.
	connectEvent, err := s.service.ConnectNode(s.db, node)
	s.Require().NoError(err)
	s.NotNil(connectEvent)

	job := &models.Job{
		Status:        models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_NO),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		NodeID: node.ID,
		JobID:  job.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)

	// Disconnect the node.
	nodeDisconnectedEvent, err := s.service.DisconnectNode(s.db, node)
	s.Require().NoError(err)
	s.Require().NoError(s.producer.Send(nodeDisconnectedEvent))
	utils.WaitForMessages(s.amqpRawConsumer, "scheduler.node.updated", "scheduler.task.updated", "scheduler.job.updated")
	// Verify that the corresponding Tasks and Jobs statuses are updated.
	s.Require().NoError(task.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Task_STATUS_NODE_FAILED), task.Status)
	s.Require().NoError(task.LoadJobFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_FINISHED), task.Job.Status)
}

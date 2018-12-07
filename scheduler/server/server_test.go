package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestDisconnectNode_UpdatesNodeJobTaskStatuses() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	s.Require().NoError(s.service.ConnectNode(node))

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

	consumer, err := utils.SetUpAMQPTestConsumer(s.amqpConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	// Disconnect the node.
	s.Require().NoError(s.service.DisconnectNode(node))
	// Wait for all the messages to be received.
	utils.WaitForMessages(consumer, "scheduler.node.updated.disconnected_at")
}

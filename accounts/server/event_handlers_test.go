package server_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events/events_proto"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestHandleNodeCreatedEvents() {
	s.service.DB = s.service.DB.LogMode(true)

	// Create a "node.created" event similar to the one that Scheduler service would create.
	nodeProto := &scheduler_proto.Node{
		Id:          42,
		Username:    "node_username",
		Name:        "node_name",
		CpuModel:    "cpu model",
		MemoryModel: "memory model",
		GpuModel:    "gpu model",
		DiskModel:   "disk model",
	}
	nodeCreatedEvent, err := events.NewEventFromPayload(nodeProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	s.Require().NoError(err)

	publisher, err := events.NewAMQPPublisher(s.service.AMQPConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)
	// Send the event.
	s.Require().NoError(publisher.Send(nodeCreatedEvent))

	// Receive Accounts service messages.
	consumer, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.AccountsAMQPExchangeName)
	s.Require().NoError(err)
	// Wait for the message be processed.
	utils.WaitForMessages(consumer, "accounts.node.created")
	// Give the service some time to commit the current transaction.
	time.Sleep(time.Millisecond*100)
	// By that time a Node is expected to be created in DB.
	node := &models.Node{}
	s.Require().NoError(node.Get(s.service.DB, nodeProto.Username, nodeProto.Name))
	s.Equal(nodeProto.Username, node.Username)
	s.Equal(nodeProto.Name, node.Name)
	s.Equal(nodeProto.CpuModel, node.CpuModel)
	s.Equal(nodeProto.MemoryModel, node.MemoryModel)
	s.Equal(nodeProto.GpuModel, node.GpuModel)
	s.Equal(nodeProto.DiskModel, node.DiskModel)
}

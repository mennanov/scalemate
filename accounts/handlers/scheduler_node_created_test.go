package handlers_test

import (
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/accounts/handlers"
	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *HandlersTestSuite) TestSchedulerNodeCreatedHandler_EventsSkipped() {
	producer := events.NewFakeProducer()

	handler := handlers.NewSchedulerNodeCreatedHandler(s.db, producer, s.logger)
	for _, event := range []*events_proto.Event{
		{
			Type: events_proto.Event_DELETED,
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{Id: 1},
			},
		},
		{
			Type: events_proto.Event_CREATED,
			Payload: &events_proto.Event_SchedulerJob{
				SchedulerJob: &scheduler_proto.Job{Id: 1},
			},
		},
		{
			Type: events_proto.Event_UPDATED,
		},
		{
			Type: events_proto.Event_UNKNOWN,
		},
	} {
		s.Require().NoError(handler.Handle(event))
		// Verify that this event has not been processed.
		s.Equal(0, len(producer.SentEvents))
	}
}

func (s *HandlersTestSuite) TestSchedulerNodeCreatedHandler_Handle() {
	producer := events.NewNatsProducer(s.conn, events.AccountsSubjectName)
	handler := handlers.NewSchedulerNodeCreatedHandler(s.db, producer, s.logger)

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
	event, err := events.NewEvent(nodeProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	s.Require().NoError(err)

	key := events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_CREATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsNode{AccountsNode: &accounts_proto.Node{Id: nodeProto.Id}},
	})
	s.Require().NoError(handler.Handle(event))
	s.Require().NoError(s.messagesHandler.ExpectMessages(key))

	// The Node is expected to be created in DB.
	node := &models.Node{}
	s.Require().NoError(node.Get(s.db, nodeProto.Username, nodeProto.Name))
	s.Equal(nodeProto.Id, node.ID)
	s.Equal(nodeProto.Username, node.Username)
	s.Equal(nodeProto.Name, node.Name)
	s.Equal(nodeProto.CpuModel, node.CpuModel)
	s.Equal(nodeProto.MemoryModel, node.MemoryModel)
	s.Equal(nodeProto.GpuModel, node.GpuModel)
	s.Equal(nodeProto.DiskModel, node.DiskModel)
}

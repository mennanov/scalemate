package handlers_test

import (
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
			Payload: &events_proto.Event_SchedulerContainer{
				SchedulerContainer: &scheduler_proto.Container{Id: 1},
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
		Id:       42,
		Username: "node_username",
		Name:     "node_name",
		Fingerprint: []byte("fingerprint"),
	}
	event := events.NewEvent(nodeProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)

	s.Require().NoError(handler.Handle(event))

	// The Node is expected to be created in DB.
	node, err := models.NodeLookUp(s.db, nodeProto.Username, nodeProto.Name)
	s.Require().NoError(err)
	s.Equal(nodeProto.Id, node.ID)
	s.Equal(nodeProto.Username, node.Username)
	s.Equal(nodeProto.Name, node.Name)
	s.Equal(nodeProto.Fingerprint, node.Fingerprint)

	// Verify that the operation is idempotent.
	s.Require().NoError(handler.Handle(event))
	var count int
	s.Require().NoError(s.db.QueryRowx("SELECT COUNT(*) FROM nodes WHERE username = $1 AND name = $2",
		node.Username, node.Name).Scan(&count))
	s.Equal(1, count)
}

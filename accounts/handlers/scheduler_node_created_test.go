package handlers_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/accounts/handlers"
	"github.com/mennanov/scalemate/accounts/models"
)

func (s *HandlersTestSuite) TestSchedulerNodeCreatedHandler_Handle() {
	handler := handlers.NewSchedulerNodeCreatedHandler(s.logger)

	nodeProto := &scheduler_proto.Node{
		Id:          42,
		Username:    "node_username",
		Name:        "node_name",
		Fingerprint: []byte("fingerprint"),
	}
	event := &events_proto.Event{
		Payload: &events_proto.Event_SchedulerNodeCreated{
			SchedulerNodeCreated: &scheduler_proto.NodeCreatedEvent{
				Node: nodeProto,
			},
		},
	}

	handlerEvents, err := handler.Handle(s.db, event)
	s.Require().NoError(err)
	// No events are expected.
	s.Nil(handlerEvents)

	// The Node is expected to be created in DB.
	node, err := models.NodeLookUp(s.db, nodeProto.Username, nodeProto.Name)
	s.Require().NoError(err)
	s.Equal(nodeProto.Id, node.Id)
	s.Equal(nodeProto.Username, node.Username)
	s.Equal(nodeProto.Name, node.Name)
	s.Equal(nodeProto.Fingerprint, node.Fingerprint)

	// Verify that the operation is idempotent.
	handlerEvents, err = handler.Handle(s.db, event)
	var count int

	s.Require().NoError(s.db.QueryRowx("SELECT COUNT(*) FROM nodes WHERE username = $1 AND name = $2",
		node.Username, node.Name).Scan(&count))
	s.Equal(1, count)
}

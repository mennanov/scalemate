package handlers_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/scheduler/handlers"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *HandlersTestSuite) TestNodeDisconnectedHandler_MarksRunningTasksAsFailed() {
	node := &models.Node{
		Username: "username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Container{}
	_, err = job.Create(s.db)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)

	now := time.Now()
	nodeUpdatedEvent, err := node.Update(s.db, map[string]interface{}{
		"disconnected_at": &now,
		"status":          utils.Enum(scheduler_proto.Node_STATUS_OFFLINE),
	})
	s.Require().NoError(err)

	producer := events.NewFakeProducer()
	handler := handlers.NewNodeDisconnectedHandler("handlerName", s.db, producer)

	s.Require().NoError(handler.Handle(nodeUpdatedEvent))
	s.Equal(1, len(producer.SentEvents))
	// Verify that the Task's status is updated.
	s.Require().NoError(task.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Task_STATUS_NODE_FAILED), task.Status)
	// Check idempotency.
	producer.SentEvents = nil
	s.Require().NoError(handler.Handle(nodeUpdatedEvent))
	s.Equal(0, len(producer.SentEvents))
}

func (s *HandlersTestSuite) TestNodeDisconnected_EventsIgnored() {
	for _, event := range []*events_proto.Event{
		{
			Type: events_proto.Event_UNKNOWN,
		},
		{
			Type: events_proto.Event_CREATED,
		},
		{
			Type: events_proto.Event_DELETED,
		},
		{
			Type: events_proto.Event_UPDATED,
			// Non-Node related event is ignored.
			Payload: &events_proto.Event_SchedulerJob{
				SchedulerJob: &scheduler_proto.Job{Id: 42},
			},
		},
		{
			Type: events_proto.Event_UPDATED,
			// Event with a nil PayloadMask is ignored.
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{Id: 42, CpuAvailable: 1},
			},
		},
		{
			Type: events_proto.Event_UPDATED,
			// Event without a "status" field in the PayloadMask is ignored.
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{Id: 42, CpuAvailable: 1},
			},
			PayloadMask: &field_mask.FieldMask{
				Paths: []string{"cpu_available"},
			},
		},
		{
			Type: events_proto.Event_UPDATED,
			// Non-disconnected Node event is ignored.
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{
					Id:     42,
					Status: scheduler_proto.Node_STATUS_SHUTTING_DOWN,
				},
			},
			PayloadMask: &field_mask.FieldMask{
				Paths: []string{"status"},
			},
		},
		{
			Type: events_proto.Event_UPDATED,
			// Non-disconnected Node event is ignored.
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{
					Id:     42,
					Status: scheduler_proto.Node_STATUS_ONLINE,
				},
			},
			PayloadMask: &field_mask.FieldMask{
				Paths: []string{"status"},
			},
		},
	} {
		producer := events.NewFakeProducer()
		handler := handlers.NewNodeDisconnectedHandler("handlerName", s.db, producer)
		s.NoError(handler.Handle(event), event)
		s.Equal(0, len(producer.SentEvents))
	}
}

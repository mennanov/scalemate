package handlers_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/scheduler/handlers"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *HandlersTestSuite) TestTaskTerminatedHandler_JobFinished() {
	node := &models.Node{
		Username:        "username",
		Name:            "node_name",
		CpuAvailable:    0,
		GpuAvailable:    0,
		MemoryAvailable: 0,
		DiskAvailable:   0,
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Container{
		Username:         "job1",
		Status:           utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		ReschedulePolicy: utils.Enum(scheduler_proto.Job_RESCHEDULE_POLICY_NO),
		CpuLimit:         1,
		GpuLimit:         1,
		MemoryLimit:      1000,
		DiskLimit:        1000,
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.db, scheduler_proto.Task_STATUS_FINISHED)
	s.Require().NoError(err)

	producer := events.NewFakeProducer()
	handler := handlers.NewTaskTerminatedHandler("handlerName", s.db, producer)

	s.Require().NoError(handler.Handle(taskUpdatedEvent))
	s.Equal(2, len(producer.SentEvents))
	// Verify that the jobScheduled now has a status "FINISHED".
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Job_STATUS_FINISHED), job.Status)
	// Verify than the Node's statistics and resources are updated.
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(float32(1), node.CpuAvailable)
	s.Equal(uint32(1), node.GpuAvailable)
	s.Equal(uint32(1000), node.MemoryAvailable)
	s.Equal(uint32(1000), node.DiskAvailable)
	s.Equal(uint64(1), node.TasksFinished)

	// Check idempotency.
	producer.SentEvents = nil
	s.Require().NoError(handler.Handle(taskUpdatedEvent))
	s.Equal(0, len(producer.SentEvents))
}

func (s *HandlersTestSuite) TestTaskTerminatedHandler_JobPending() {
	node := &models.Node{
		Username:     "username",
		Name:         "node_name",
		CpuAvailable: 0,
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Container{
		Username:         "job1",
		Status:           utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		ReschedulePolicy: utils.Enum(scheduler_proto.Job_RESCHEDULE_POLICY_ON_FAILURE),
		CpuLimit:         1,
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
		Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)

	taskUpdatedEvent, err := task.UpdateStatus(s.db, scheduler_proto.Task_STATUS_NODE_FAILED)
	s.Require().NoError(err)
	producer := events.NewFakeProducer()
	handler := handlers.NewTaskTerminatedHandler("handlerName", s.db, producer)

	s.Require().NoError(handler.Handle(taskUpdatedEvent))
	s.Equal(2, len(producer.SentEvents))

	// Verify that the job now has a status "PENDING".
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Job_STATUS_PENDING), job.Status)
	// Verify than the Node's statistics and resources are updated.
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(float32(1), node.CpuAvailable)
	s.Equal(uint64(1), node.TasksFailed)

	// Check idempotency.
	producer.SentEvents = nil
	s.Require().NoError(handler.Handle(taskUpdatedEvent))
	s.Equal(0, len(producer.SentEvents))
}

func (s *HandlersTestSuite) TestTaskTerminatedHandler_EventsIgnored() {
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
			PayloadMask: &field_mask.FieldMask{
				Paths: []string{"status"},
			},
			// Non-Task related event is ignored.
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{Id: 42},
			},
		},
		{
			Type: events_proto.Event_UPDATED,
			PayloadMask: &field_mask.FieldMask{
				Paths: []string{"status"},
			},
			// Non-terminated Task event is ignored.
			Payload: &events_proto.Event_SchedulerTask{
				SchedulerTask: &scheduler_proto.Task{
					Id:     42,
					Status: scheduler_proto.Task_STATUS_RUNNING,
				},
			},
		},
	} {
		producer := events.NewFakeProducer()
		handler := handlers.NewTaskTerminatedHandler("handlerName", s.db, producer)
		s.NoError(handler.Handle(event))
		s.Equal(0, len(producer.SentEvents))
	}
}

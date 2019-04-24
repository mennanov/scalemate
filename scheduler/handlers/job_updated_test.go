package handlers_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/scheduler/handlers"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *HandlersTestSuite) TestJobUpdatedHandler_SchedulesPendingJob() {
	node := &models.Node{
		Username:        "node_username",
		Name:            "node_name",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    4,
		CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 8000,
		DiskCapacity:    20000,
		DiskAvailable:   20000,
		DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Container{
		CpuLimit:    2,
		MemoryLimit: 4000,
		DiskLimit:   10000,
		Status:      utils.Enum(scheduler_proto.Job_STATUS_NEW),
	}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	jobUpdatedEvent, err := job.UpdateStatus(s.db, scheduler_proto.Job_STATUS_PENDING)
	s.Require().NoError(err)

	producer := events.NewFakeProducer()
	handler := handlers.NewJobUpdatedHandler("handlerName", s.db, producer, s.logger)

	s.Require().NoError(handler.Handle(jobUpdatedEvent))
	// Verify that the corresponding Task is created.
	s.Require().NoError(job.LoadTasksFromDB(s.db))
	s.Require().Len(job.Tasks, 1)
	s.Equal(job.ID, job.Tasks[0].JobID)
	// Verify that the Container status is now SCHEDULED.
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED), job.Status)
	// Verify that the Node's resources are updated.
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(float32(2), node.CpuAvailable)
	s.Equal(uint32(4000), node.MemoryAvailable)
	s.Equal(uint32(10000), node.DiskAvailable)
	// Check idempotency.
	s.Require().NoError(handler.Handle(jobUpdatedEvent))
	s.Require().NoError(job.LoadTasksFromDB(s.db))
	s.Require().Len(job.Tasks, 1)
	s.Equal(job.ID, job.Tasks[0].JobID)
}

func (s *HandlersTestSuite) TestJobUpdatedHandler_EventsIgnored() {
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
			// Non-Container related event is ignored.
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{Id: 42},
			},
		},
		{
			Type: events_proto.Event_UPDATED,
			// Non-pending Container event is ignored.
			Payload: &events_proto.Event_SchedulerJob{
				SchedulerJob: &scheduler_proto.Job{
					Id:     42,
					Status: scheduler_proto.Job_STATUS_FINISHED,
				},
			},
		},
	} {
		producer := events.NewFakeProducer()
		handler := handlers.NewJobUpdatedHandler("handlerName", s.db, producer, s.logger)
		s.NoError(handler.Handle(event))
		s.Equal(0, len(producer.SentEvents))
	}
}

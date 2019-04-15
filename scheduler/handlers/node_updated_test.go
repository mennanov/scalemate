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

func (s *HandlersTestSuite) TestNodeUpdatedHandler_SchedulesPendingJobs() {
	node := &models.Node{
		Username:        "username",
		Name:            "node_name",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    4,
		CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		MemoryCapacity:  32000,
		MemoryAvailable: 16000,
		DiskCapacity:    64000,
		DiskAvailable:   28000,
		DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	// All these Jobs are expected to fit into the Node above.
	jobs := []*models.Job{
		{
			Username:    "job1",
			Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    1,
			CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
			MemoryLimit: 4000,
			DiskLimit:   7000,
			DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		},
		{
			Username:    "job2",
			Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    1,
			CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
			MemoryLimit: 4000,
			DiskLimit:   7000,
			DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		},
	}

	var jobIds []uint64
	for _, job := range jobs {
		_, err := job.Create(s.db)
		s.Require().NoError(err)
		jobIds = append(jobIds, job.ID)
	}
	// Manually create a Node connected event.
	mask := &field_mask.FieldMask{
		Paths: []string{"status", "connected_at"},
	}
	nodeProto, err := node.ToProto(mask)
	s.Require().NoError(err)
	eventProto, err := events.NewEvent(nodeProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER, mask)
	s.Require().NoError(err)

	producer := events.NewFakeProducer()
	handler := handlers.NewNodeUpdatedHandler("handlerName", s.db, producer, s.logger)
	s.Require().NoError(handler.Handle(eventProto))
	s.Equal(3*len(jobs), len(producer.SentEvents))

	// Reload Node from DB.
	s.Require().NoError(node.LoadFromDB(s.db))
	// Jobs are expected to acquire half of the Node's resources.
	s.Equal(float32(2), node.CpuAvailable)
	s.Equal(uint32(14000), node.DiskAvailable)
	s.Equal(uint32(0), node.GpuAvailable)
	s.Equal(uint32(8000), node.MemoryAvailable)
	// Verify that Jobs now have a status "SCHEDULED".
	for _, job := range jobs {
		s.Require().NoError(job.LoadFromDB(s.db))
		s.Equal(utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED), job.Status)
		s.Require().NoError(job.LoadTasksFromDB(s.db))
		s.Equal(1, len(job.Tasks))
		s.Equal(node.ID, job.Tasks[0].NodeID)
	}
	// Check idempotency.
	producer.SentEvents = nil
	s.Require().NoError(handler.Handle(eventProto))
	s.Equal(0, len(producer.SentEvents))
	// Check that the Node's resources are intact.
	s.Equal(float32(2), node.CpuAvailable)
	s.Equal(uint32(14000), node.DiskAvailable)
	s.Equal(uint32(0), node.GpuAvailable)
	s.Equal(uint32(8000), node.MemoryAvailable)
}

func (s *HandlersTestSuite) TestNodeUpdatedHandler_EventsIgnored() {
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
			// Non-connected Node event is ignored.
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
			// Non-connected Node event is ignored.
			Payload: &events_proto.Event_SchedulerNode{
				SchedulerNode: &scheduler_proto.Node{
					Id:     42,
					Status: scheduler_proto.Node_STATUS_OFFLINE,
				},
			},
			PayloadMask: &field_mask.FieldMask{
				Paths: []string{"status"},
			},
		},
	} {
		producer := events.NewFakeProducer()
		handler := handlers.NewNodeUpdatedHandler("handlerName", s.db, producer, s.logger)
		s.NoError(handler.Handle(event), event)
		s.Equal(0, len(producer.SentEvents))
	}
}

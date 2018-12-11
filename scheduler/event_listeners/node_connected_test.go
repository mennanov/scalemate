package event_listeners_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/scheduler/event_listeners"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *EventListenersTestSuite) TestNodeConnectedHandler_SchedulesPendingJobsOnTheNode() {
	node := &models.Node{
		Username:        "username",
		Name:            "node_name",
		Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    2,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		MemoryCapacity:  32000,
		MemoryAvailable: 16000,
		DiskCapacity:    64000,
		DiskAvailable:   28000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	// All these Jobs are expected to fit into the Node above.
	jobs := []*models.Job{
		{
			Username:    "job1",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    1,
			CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
			MemoryLimit: 8000,
			DiskLimit:   14000,
			DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		},
		{
			Username:    "job2",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    1,
			CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
			MemoryLimit: 8000,
			DiskLimit:   14000,
			DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		},
	}

	var jobIds []uint64
	for _, job := range jobs {
		_, err := job.Create(s.service.DB)
		s.Require().NoError(err)
		jobIds = append(jobIds, job.ID)
	}
	mask := &field_mask.FieldMask{
		Paths: []string{"status", "connected_at"},
	}
	nodeProto, err := node.ToProto(mask)
	s.Require().NoError(err)
	eventProto, err := events.NewEventFromPayload(nodeProto, events_proto.Event_UPDATED, events_proto.Service_SCHEDULER, mask)
	s.Require().NoError(err)
	// Run the handler.
	s.Require().NoError(event_listeners.NodeConnectedAMQPEventListener.Handler(s.service, eventProto))

	// Reload Node from DB.
	s.Require().NoError(node.LoadFromDB(s.service.DB))
	// Jobs are expected to acquire all Node's resources.
	s.Equal(float32(0), node.CpuAvailable)
	s.Equal(uint32(0), node.DiskAvailable)
	s.Equal(uint32(0), node.GpuAvailable)
	s.Equal(uint32(0), node.MemoryAvailable)
	// Verify that Jobs now have a status "SCHEDULED".
	for _, job := range jobs {
		s.Require().NoError(job.LoadFromDB(s.service.DB))
		s.Equal(models.Enum(scheduler_proto.Job_STATUS_SCHEDULED), job.Status)
	}
}

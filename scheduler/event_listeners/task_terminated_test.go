package event_listeners_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/event_listeners"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *EventListenersTestSuite) TestTaskTerminatedHandler_TaskFinished_JobTerminated_PendingJobsScheduledOnTheNode() {
	node := &models.Node{
		Username:        "username",
		Name:            "node_name",
		Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    2,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		GpuCapacity:     2,
		GpuAvailable:    2,
		GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
		GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
		MemoryCapacity:  32000,
		MemoryAvailable: 16000,
		DiskCapacity:    64000,
		DiskAvailable:   28000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	jobPending := &models.Job{
		Username:    "job1",
		Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1,
		CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		GpuLimit:    1,
		GpuClass:    models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
		MemoryLimit: 8000,
		DiskLimit:   14000,
		DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
	}
	_, err = jobPending.Create(s.service.DB)
	s.Require().NoError(err)

	jobScheduled := &models.Job{
		Username:    "job2",
		Status:      models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		CpuLimit:    1,
		CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		GpuLimit:    1,
		GpuClass:    models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
		MemoryLimit: 8000,
		DiskLimit:   14000,
		DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
	}
	_, err = jobScheduled.Create(s.service.DB)
	s.Require().NoError(err)

	taskRunning := &models.Task{
		JobID:  jobScheduled.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = taskRunning.Create(s.service.DB)
	s.Require().NoError(err)

	taskUpdatedEvent, err := taskRunning.UpdateStatus(s.service.DB, scheduler_proto.Task_STATUS_FINISHED)

	s.Require().NoError(event_listeners.TaskTerminatedAMQPEventListener.Handler(s.service, taskUpdatedEvent))

	s.Len(s.service.Publisher.(*utils.FakePublisher).SentEvents, 4)

	// Verify that the jobScheduled now has a status "FINISHED".
	s.Require().NoError(jobScheduled.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_FINISHED), jobScheduled.Status)

	// Verify that the jobPending now has a status "SCHEDULED".
	s.Require().NoError(jobPending.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_SCHEDULED), jobPending.Status)

	// Verify that the Node's resources are updated for the scheduled Job jobPending.
	s.Require().NoError(node.LoadFromDB(s.service.DB))
	s.Equal(float32(1), node.CpuAvailable)
	s.Equal(uint32(14000), node.DiskAvailable)
	s.Equal(uint32(1), node.GpuAvailable)
	s.Equal(uint32(8000), node.MemoryAvailable)
}

package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestHandleJobPending_JobScheduledToAvailableNode() {
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

	job := &models.Job{
		CpuLimit:    4,
		MemoryLimit: 8000,
		DiskLimit:   20000,
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
	}
	jobCreatedEvent, err := job.Create(s.db)
	s.Require().NoError(err)

	s.Require().NoError(s.service.HandleJobPending(jobCreatedEvent))
	// Verify that the corresponding Task is created.
	s.Require().NoError(job.LoadTasksFromDB(s.db))
	s.Require().Len(job.Tasks, 1)
	s.Equal(job.ID, job.Tasks[0].JobID)
	// Verify that the Job status is now SCHEDULED.
	s.Require().NoError(job.LoadFromDB(s.db))
	s.Equal(utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED), job.Status)
	// Verify that the Node's resources are updated.
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(float32(0), node.CpuAvailable)
	s.Equal(uint32(0), node.MemoryAvailable)
	s.Equal(uint32(0), node.DiskAvailable)
}

package models_test

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ModelsTestSuite) TestNode_FromProtoToProto() {
	testCases := []struct {
		mask      fieldmask_utils.FieldFilter
		nodeProto *scheduler_proto.Node
	}{
		{
			mask: fieldmask_utils.Mask{},
			nodeProto: &scheduler_proto.Node{
				Id:              1,
				Username:        "username",
				Name:            "node_name",
				Status:          scheduler_proto.Node_STATUS_ONLINE,
				CpuCapacity:     4,
				CpuAvailable:    2.5,
				CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				CpuClassMin:     scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
				MemoryCapacity:  16000,
				MemoryAvailable: 10000,
				GpuCapacity:     4,
				GpuAvailable:    2,
				GpuClass:        scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				GpuClassMin:     scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
				DiskCapacity:    20000,
				DiskAvailable:   15000,
				DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_SSD,
				DiskClassMin:    scheduler_proto.DiskClass_DISK_CLASS_HDD,
				ConnectedAt: &timestamp.Timestamp{
					Seconds: time.Now().Unix(),
				},
				DisconnectedAt: &timestamp.Timestamp{
					Seconds: time.Now().Unix(),
				},
				ScheduledAt: &timestamp.Timestamp{
					Seconds: time.Now().Unix(),
				},
				CreatedAt: &timestamp.Timestamp{
					Seconds: time.Now().Unix(),
					Nanos:   4000,
				},
				UpdatedAt: &timestamp.Timestamp{
					Seconds: time.Now().Unix(),
				},
			},
		},
		{
			mask: fieldmask_utils.MaskInverse{"CreatedAt": nil, "UpdatedAt": nil},
			nodeProto: &scheduler_proto.Node{
				Id:       2,
				Username: "username2",
				Name:     "node_name2",
			},
		},
	}
	for _, testCase := range testCases {
		node := &models.Node{}
		err := node.FromProto(testCase.nodeProto)
		s.Require().NoError(err)
		nodeProto, err := node.ToProto(nil)
		s.Require().NoError(err)
		s.Equal(testCase.nodeProto, nodeProto)

		// Create the node in DB.
		_, err = node.Create(s.db)
		s.Require().NoError(err)

		// Retrieve the same Node from DB.
		nodeFromDB := &models.Node{}
		s.db.First(nodeFromDB, node.ID)
		nodeFromDBProto, err := nodeFromDB.ToProto(nil)
		s.Require().NoError(err)

		nodeFromDBProtoFiltered := &scheduler_proto.Node{}
		s.Require().NoError(fieldmask_utils.StructToStruct(testCase.mask, nodeFromDBProto, nodeFromDBProtoFiltered))

		s.Equal(testCase.nodeProto, nodeFromDBProtoFiltered)
	}
}

func (s *ModelsTestSuite) TestNode_AllocateJobResources() {
	scheduledAtOld := time.Date(2018, 10, 30, 21, 59, 0, 0, time.FixedZone("t", 0))
	node := &models.Node{
		Username:        "node_owner",
		Name:            "node1",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuAvailable:    3,
		MemoryAvailable: 6000,
		GpuAvailable:    4,
		DiskAvailable:   10000,
		ScheduledAt:     &scheduledAtOld,
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job1 := &models.Job{
		CpuLimit:    2,
		MemoryLimit: 4000,
		GpuLimit:    4,
		DiskLimit:   7000,
	}
	event1, err := node.AllocateJobResources(s.db, job1)
	s.Require().NoError(err)
	s.NotNil(event1)
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(float32(1), node.CpuAvailable)
	s.Equal(uint32(2000), node.MemoryAvailable)
	s.Equal(uint32(0), node.GpuAvailable)
	s.Equal(uint32(3000), node.DiskAvailable)
	s.True(node.ScheduledAt.After(scheduledAtOld))

	// Second consecutive job takes all remaining resources.
	job2 := &models.Job{
		CpuLimit:    1,
		MemoryLimit: 2000,
		GpuLimit:    0,
		DiskLimit:   3000,
	}
	event2, err := node.AllocateJobResources(s.db, job2)
	s.Require().NoError(err)
	s.NotNil(event2)
	s.NotEqual(event1, event2)
	s.Require().NoError(node.LoadFromDB(s.db))
	s.Equal(float32(0), node.CpuAvailable)
	s.Equal(uint32(0), node.MemoryAvailable)
	s.Equal(uint32(0), node.GpuAvailable)
	s.Equal(uint32(0), node.DiskAvailable)
}

func (s *ModelsTestSuite) TestNode_AllocateJobResources_Fails() {
	scheduledAtOld := time.Date(2018, 10, 30, 21, 59, 0, 0, time.FixedZone("t", 0))
	node := &models.Node{
		CpuAvailable:    3,
		MemoryAvailable: 6000,
		GpuAvailable:    4,
		DiskAvailable:   10000,
		ScheduledAt:     &scheduledAtOld,
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	s.T().Run("CpuLimit fails", func(t *testing.T) {
		job := &models.Job{
			CpuLimit: 3.5,
		}
		event, err := node.AllocateJobResources(s.db, job)
		s.Error(err)
		s.Nil(event)
	})

	s.T().Run("GpuLimit fails", func(t *testing.T) {
		job := &models.Job{
			GpuLimit: 5,
		}
		event, err := node.AllocateJobResources(s.db, job)
		s.Error(err)
		s.Nil(event)
	})

	s.T().Run("MemoryLimit fails", func(t *testing.T) {
		job := &models.Job{
			MemoryLimit: 7000,
		}
		event, err := node.AllocateJobResources(s.db, job)
		s.Error(err)
		s.Nil(event)
	})

	s.T().Run("DiskLimit fails", func(t *testing.T) {
		job := &models.Job{
			DiskLimit: 20000,
		}
		event, err := node.AllocateJobResources(s.db, job)
		s.Error(err)
		s.Nil(event)
	})
}

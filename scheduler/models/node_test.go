package models_test

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
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
		_, err = node.Create(s.gormDB)
		s.Require().NoError(err)

		// Retrieve the same Node from DB.
		nodeFromDB := &models.Node{}
		s.gormDB.First(nodeFromDB, node.ID)
		nodeFromDBProto, err := nodeFromDB.ToProto(nil)
		s.Require().NoError(err)

		nodeFromDBProtoFiltered := &scheduler_proto.Node{}
		s.Require().NoError(fieldmask_utils.StructToStruct(testCase.mask, nodeFromDBProto, nodeFromDBProtoFiltered))

		s.Equal(testCase.nodeProto, nodeFromDBProtoFiltered)
	}
}

func (s *ModelsTestSuite) TestNode_AllocateJobResources() {
	node := &models.Node{
		CpuAvailable:    3,
		MemoryAvailable: 6000,
		GpuAvailable:    4,
		DiskAvailable:   10000,
	}

	job := &models.Container{
		CpuLimit:    2,
		MemoryLimit: 4000,
		GpuLimit:    4,
		DiskLimit:   7000,
	}
	updates, err := node.AllocateContainerLimit(job)
	s.Require().NoError(err)
	s.Equal(map[string]interface{}{
		"cpu_available":    node.CpuAvailable - job.CpuLimit,
		"memory_available": node.MemoryAvailable - job.MemoryLimit,
		"gpu_available":    node.GpuAvailable - job.GpuLimit,
		"disk_available":   node.DiskAvailable - job.DiskLimit,
	}, updates)
}

func (s *ModelsTestSuite) TestNode_AllocateJobResources_Fails() {
	node := &models.Node{
		CpuAvailable:    3,
		MemoryAvailable: 6000,
		GpuAvailable:    4,
		DiskAvailable:   10000,
	}

	s.T().Run("CpuLimit fails", func(t *testing.T) {
		job := &models.Container{
			CpuLimit: 3.5,
		}
		updates, err := node.AllocateContainerLimit(job)
		s.Error(err)
		s.Nil(updates)
	})

	s.T().Run("GpuLimit fails", func(t *testing.T) {
		job := &models.Container{
			GpuLimit: 5,
		}
		updates, err := node.AllocateContainerLimit(job)
		s.Error(err)
		s.Nil(updates)
	})

	s.T().Run("MemoryLimit fails", func(t *testing.T) {
		job := &models.Container{
			MemoryLimit: 7000,
		}
		updates, err := node.AllocateContainerLimit(job)
		s.Error(err)
		s.Nil(updates)
	})

	s.T().Run("DiskLimit fails", func(t *testing.T) {
		job := &models.Container{
			DiskLimit: 20000,
		}
		updates, err := node.AllocateContainerLimit(job)
		s.Error(err)
		s.Nil(updates)
	})
}

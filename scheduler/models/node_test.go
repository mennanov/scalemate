package models_test

import (
	"testing"
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ModelsTestSuite) TestNode_FromProtoToProto() {
	testCases := []struct {
		mask      string
		nodeProto *scheduler_proto.Node
	}{
		{
			mask: "username,status,cpu_capacity,cpu_available,cpu_class,cpu_class_min,memory_capacity," +
				"memory_available,gpu_capacity,gpu_available,gpu_class,gpu_class_min,disk_capacity,disk_available," +
				"disk_class,disk_class_min,connected_at,disconnected_at,scheduled_at,created_at,updated_at",
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
			mask: "id,username",
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
		// Create the node in DB.
		_, err = node.Create(s.db)
		s.Require().NoError(err)
		// Retrieve the same node from DB.
		nodeFromDb := &models.Node{}
		s.db.First(nodeFromDb, node.ID)
		p2, err := nodeFromDb.ToProto(nil)
		s.Require().NoError(err)

		expected := make(map[string]interface{})
		mask := fieldmask_utils.MaskFromString(testCase.mask)
		err = fieldmask_utils.StructToMap(mask, testCase.nodeProto, expected, generator.CamelCase, stringEye)
		s.Require().NoError(err)

		actual := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, p2, actual, generator.CamelCase, stringEye)
		s.Require().NoError(err)

		s.Equal(expected, actual)
	}
}

func (s *ModelsTestSuite) TestNode_FindSuitableJobs() {
	node := &models.Node{
		Username:        "node_owner",
		Name:            "node1",
		Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    3.5,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuModel:        "Intel Core i7 @ 2.20GHz",
		MemoryCapacity:  8000,
		MemoryAvailable: 6000,
		MemoryModel:     "DDR3-1600MHz",
		GpuCapacity:     4,
		GpuAvailable:    4,
		GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuModel:        "Intel Iris Pro 1536MB",
		DiskCapacity:    20000,
		DiskAvailable:   10000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskModel:       "251GB APPLE SSD SM0256F",
		Labels:          []string{"special_promo_label", "sale20"},
		ConnectedAt:     time.Now(),
	}
	node.Create(s.db)

	jobs := []*models.Job{
		// This Job satisfies criteria.
		{
			Username:       "job1_username",
			Status:         models.Enum(scheduler_proto.Job_STATUS_PENDING),
			DockerImage:    "postgres:latest",
			CpuLimit:       1,
			CpuClass:       models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryLimit:    2000,
			GpuLimit:       2,
			GpuClass:       models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
			DiskLimit:      2000,
			DiskClass:      models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			CpuLabels:      []string{"Intel Core i7 @ 2.20GHz", "Intel Core i5 @ 2.20GHz"},
			GpuLabels:      []string{"Intel Iris Pro 1536MB"},
			DiskLabels:     []string{"251GB APPLE SSD SM0256F"},
			MemoryLabels:   []string{"DDR3-1600MHz"},
			UsernameLabels: []string{"node_owner"},
			NameLabels:     []string{"node1", "node2"},
			OtherLabels:    []string{"sale20"},
		},
		// This Job also satisfies criteria.
		{
			Username:    "job2_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			DockerImage: "postgres:latest",
			CpuLimit:    1,
			CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryLimit: 2000,
			GpuLimit:    2,
			GpuClass:    models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
			DiskLimit:   2000,
			DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		},
		// This Job also satisfies criteria: (No hardware classes specified).
		{
			Username:    "job3_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			DockerImage: "postgres:latest",
			CpuLimit:    3.5,
			MemoryLimit: 2000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username:    "job4_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			DockerImage: "postgres:latest",
			// CpuLimit is too high.
			CpuLimit:    4,
			MemoryLimit: 2000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username:    "job5_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			DockerImage: "postgres:latest",
			CpuLimit:    3,
			// Memory limit is too high.
			MemoryLimit: 8000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username:    "job6_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			DockerImage: "postgres:latest",
			CpuLimit:    3,
			MemoryLimit: 4000,
			// GpuLimit is too high.
			GpuLimit:  6,
			DiskLimit: 2000,
		},
		{
			Username:    "job7_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			DockerImage: "postgres:latest",
			CpuLimit:    3,
			MemoryLimit: 4000,
			GpuLimit:    2,
			// DiskLimit is too high.
			DiskLimit: 20000,
		},
	}
	for _, job := range jobs {
		_, err := job.Create(s.db)
		s.Require().NoError(err)
	}

	actualJobs, err := node.FindSuitableJobs(s.db)
	s.Require().NoError(err)
	actualJobsUsernames := make(map[string]struct{}, 0)
	for _, job := range actualJobs {
		actualJobsUsernames[job.Username] = struct{}{}
	}
	s.Equal(map[string]struct{}{"job1_username": {}, "job2_username": {}, "job3_username": {}}, actualJobsUsernames)
}

func (s *ModelsTestSuite) TestNode_AllocateJobResources() {
	scheduledAtOld := time.Date(2018, 10, 30, 21, 59, 0, 0, time.FixedZone("t", 0))
	node := &models.Node{
		Username:        "node_owner",
		Name:            "node1",
		Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuAvailable:    3,
		MemoryAvailable: 6000,
		GpuAvailable:    4,
		DiskAvailable:   10000,
		ScheduledAt:     scheduledAtOld,
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
		ScheduledAt:     scheduledAtOld,
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

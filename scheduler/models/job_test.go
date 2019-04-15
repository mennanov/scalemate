package models_test

import (
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ModelsTestSuite) TestJob_FromProto_ToProto() {
	now := time.Now().Unix()
	testCases := []struct {
		jobProto *scheduler_proto.Job
		mask     fieldmask_utils.FieldFilter
	}{
		{
			jobProto: &scheduler_proto.Job{
				Id:          0,
				Username:    "username",
				Status:      scheduler_proto.Job_STATUS_CANCELLED,
				CpuLimit:    4,
				CpuClass:    scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				MemoryLimit: 2048,
				GpuLimit:    2,
				GpuClass:    scheduler_proto.GPUClass_GPU_CLASS_PRO,
				DiskLimit:   10240,
				DiskClass:   scheduler_proto.DiskClass_DISK_CLASS_HDD,
				RunConfig: &scheduler_proto.Job_RunConfig{
					Image:   "nginx:latest",
					Command: "nginx --daemon=false",
					Ports:   map[uint32]uint32{8080: 80, 4443: 443},
					Volumes: map[string]string{
						"./nginx.conf": "/etc/nginx/nginx.conf",
					},
				},
				CreatedAt: &timestamp.Timestamp{
					Seconds: now,
				},
				UpdatedAt: &timestamp.Timestamp{
					Seconds: now,
				},
				ReschedulePolicy: scheduler_proto.Job_RESCHEDULE_POLICY_ON_FAILURE,
				CpuLabels:        []string{"Intel Core i7 @ 2.20GHz", "Intel Core i5 @ 2.20GHz"},
				GpuLabels:        []string{"Intel Iris Pro 1536MB", "Intel Iris Pro 2000MB"},
				DiskLabels:       []string{"251GB APPLE SSD SM0256F"},
				MemoryLabels:     []string{"DDR3-1600MHz"},
				UsernameLabels:   []string{"username1", "username2"},
				NameLabels:       []string{"node1", "node2"},
				OtherLabels:      []string{"Europe/Samara", "USA/SF"},
			},
			mask: fieldmask_utils.MaskInverse{"Id": nil},
		},
		{
			jobProto: &scheduler_proto.Job{
				Username: "username",
			},
			mask: fieldmask_utils.MaskFromString("Username"),
		},
	}

	for _, testCase := range testCases {
		job := &models.Job{}
		err := job.FromProto(testCase.jobProto)
		s.Require().NoError(err)
		jobProto, err := job.ToProto(nil)
		s.Require().NoError(err)
		s.Equal(testCase.jobProto, jobProto)

		// Create the job in DB to verify that everything is persisted.
		_, err = job.Create(s.db)
		s.Require().NoError(err)

		// Retrieve the same job from DB.
		jobFromDB := &models.Job{}
		s.db.First(jobFromDB, job.ID)
		jobFromDBProto, err := jobFromDB.ToProto(nil)
		s.Require().NoError(err)

		// Proto message for the Job from DB differs from the original one in the test case as there are some values
		// added to the Job when it's saved.
		jobFromDBProtoFiltered := &scheduler_proto.Job{}
		err = fieldmask_utils.StructToStruct(testCase.mask, jobFromDBProto, jobFromDBProtoFiltered)
		s.Require().NoError(err)
		s.Equal(testCase.jobProto, jobFromDBProtoFiltered)
	}
}

func (s *ModelsTestSuite) TestJob_CreateTask() {
	job := &models.Job{
		Username:    "username",
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1.5,
		CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit: 1000,
		GpuLimit:    2,
		GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskLimit:   1000,
		DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now().UTC()

	node := &models.Node{
		Username:        "node_owner",
		Name:            "something",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    2.5,
		CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 4000,
		GpuCapacity:     4,
		GpuAvailable:    2,
		GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   10000,
		DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		ConnectedAt:     &now,
	}
	_, err = node.Create(s.db)
	s.Require().NoError(err)

	schedulingEvents, err := job.CreateTask(s.db, node)
	s.Require().NoError(err)
	s.Equal(3, len(schedulingEvents))
	eventPayload, err := events.NewModelProtoFromEvent(schedulingEvents[0])
	s.Require().NoError(err)
	s.Equal(node.ID, eventPayload.(*scheduler_proto.Task).NodeId)
	nodeFromDB := &models.Node{Model: utils.Model{ID: node.ID}}
	s.Require().NoError(nodeFromDB.LoadFromDB(s.db))
	s.Equal(float32(1), nodeFromDB.CpuAvailable)
	s.Equal(uint32(3000), nodeFromDB.MemoryAvailable)
	s.Equal(uint32(0), nodeFromDB.GpuAvailable)
	s.Equal(uint32(9000), nodeFromDB.DiskAvailable)
}

func (s *ModelsTestSuite) TestJob_FindSuitableNode_OneAvailable() {
	job := &models.Job{
		Username:    "username",
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1.5,
		CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit: 1000,
		GpuLimit:    2,
		GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:   1000,
		DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
		// Offline node.
		{
			Username: "node_owner",
			Name:     "node2",
			Status:   utils.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
		// Does not satisfy by GpuClass.
		{
			Username:        "node_owner",
			Name:            "node3",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 7000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	node, err := job.FindSuitableNode(s.db)
	s.Require().NoError(err)
	s.Equal("node1", node.Name)
}

func (s *ModelsTestSuite) TestJob_FindSuitableNode_BreakTieCPU() {
	job := &models.Job{
		Username:    "username",
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1.5,
		CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit: 1000,
		GpuLimit:    2,
		GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:   1000,
		DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    3,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	node, err := job.FindSuitableNode(s.db)
	s.Require().NoError(err)
	s.Equal("node2", node.Name)
}

func (s *ModelsTestSuite) TestJob_FindSuitableNode_BreakTieMemory() {
	job := &models.Job{
		Username:    "username",
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1.5,
		CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit: 1000,
		GpuLimit:    2,
		GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:   1000,
		DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	node, err := job.FindSuitableNode(s.db)
	s.Require().NoError(err)
	s.Equal("node1", node.Name)
}

func (s *ModelsTestSuite) TestJob_FindSuitableNode_BreakTieDisk() {
	job := &models.Job{
		Username:    "username",
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1.5,
		CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit: 1000,
		GpuLimit:    2,
		GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:   1000,
		DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   15000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	node, err := job.FindSuitableNode(s.db)
	s.Require().NoError(err)
	s.Equal("node2", node.Name)
}

func (s *ModelsTestSuite) TestJob_FindSuitableNode_BreakTieGPU() {
	job := &models.Job{
		Username:    "username",
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1.5,
		CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit: 1000,
		GpuLimit:    2,
		GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:   1000,
		DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    3,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	node, err := job.FindSuitableNode(s.db)
	s.Require().NoError(err)
	s.Equal("node1", node.Name)
}

func (s *ModelsTestSuite) TestJob_FindSuitableNode_BreakTieScheduledAt() {
	job := &models.Job{
		Username:    "username",
		Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1.5,
		CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit: 1000,
		GpuLimit:    2,
		GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:   1000,
		DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()
	earlier1Min := now.Add(-time.Minute)
	earlier2Min := now.Add(-time.Minute * 2)

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    3,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &now,
			ScheduledAt:     &now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    3,
			GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     &earlier2Min,
			ScheduledAt:     &earlier1Min,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	node, err := job.FindSuitableNode(s.db)
	s.Require().NoError(err)
	s.Equal("node2", node.Name)
}

func (s *ModelsTestSuite) TestJobs_FindPendingForNode() {
	now := time.Now()

	node := &models.Node{
		Username:        "node_owner",
		Name:            "node1",
		Status:          utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    3.5,
		CpuClass:        utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuModel:        "Intel Core i7 @ 2.20GHz",
		MemoryCapacity:  8000,
		MemoryAvailable: 6000,
		MemoryModel:     "DDR3-1600MHz",
		GpuCapacity:     4,
		GpuAvailable:    4,
		GpuClass:        utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		GpuClassMin:     utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuModel:        "Intel Iris Pro 1536MB",
		DiskCapacity:    20000,
		DiskAvailable:   10000,
		DiskClass:       utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskModel:       "251GB APPLE SSD SM0256F",
		Labels:          []string{"special_promo_label", "sale20"},
		ConnectedAt:     &now,
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	jobs := []*models.Job{
		// This Job satisfies criteria.
		{
			Username:       "job1_username",
			Status:         utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:       1,
			CpuClass:       utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryLimit:    2000,
			GpuLimit:       2,
			GpuClass:       utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
			DiskLimit:      2000,
			DiskClass:      utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
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
			Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    1,
			CpuClass:    utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryLimit: 2000,
			GpuLimit:    2,
			GpuClass:    utils.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
			DiskLimit:   2000,
			DiskClass:   utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		},
		// This Job also satisfies criteria: (No hardware classes specified).
		{
			Username:    "job3_username",
			Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    3.5,
			MemoryLimit: 2000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username: "job4_username",
			Status:   utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			// CpuLimit is too high.
			CpuLimit:    4,
			MemoryLimit: 2000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username: "job5_username",
			Status:   utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit: 3,
			// Memory limit is too high.
			MemoryLimit: 8000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username:    "job6_username",
			Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    3,
			MemoryLimit: 4000,
			// GpuLimit is too high.
			GpuLimit:  6,
			DiskLimit: 2000,
		},
		{
			Username:    "job7_username",
			Status:      utils.Enum(scheduler_proto.Job_STATUS_PENDING),
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

	var pendingJobs models.Jobs
	err = pendingJobs.FindPendingForNode(s.db, node)
	s.Require().NoError(err)
	pendingJobsUsernames := make(map[string]struct{})
	for _, job := range pendingJobs {
		pendingJobsUsernames[job.Username] = struct{}{}
	}
	s.Equal(map[string]struct{}{"job1_username": {}, "job2_username": {}, "job3_username": {}}, pendingJobsUsernames)
}

func (s *ModelsTestSuite) TestJob_LoadTasksFromDB() {
	node := &models.Node{}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	job := &models.Job{}
	_, err = job.Create(s.db)
	s.Require().NoError(err)

	tasks := []*models.Task{
		{
			NodeID: node.ID,
			JobID:  job.ID,
		},
		{
			NodeID: node.ID,
			JobID:  job.ID,
		},
	}

	for _, task := range tasks {
		_, err = task.Create(s.db)
		s.Require().NoError(err)
	}

	s.Equal(0, len(job.Tasks))
	s.Require().NoError(job.LoadTasksFromDB(s.db))
	s.Equal(len(tasks), len(job.Tasks))
	// Verify that the method is idempotent.
	s.Require().NoError(job.LoadTasksFromDB(s.db))
	s.Equal(len(tasks), len(job.Tasks))
}

func (s *ModelsTestSuite) TestJob_UpdateStatus() {
	for statusFrom, statusesTo := range models.JobStatusTransitions {
		for _, statusTo := range statusesTo {
			job := &models.Job{Status: utils.Enum(statusFrom)}
			_, err := job.Create(s.db)
			s.Require().NoError(err)
			s.Nil(job.UpdatedAt)

			_, err = job.UpdateStatus(s.db, statusTo)
			s.Require().NoError(err)
			s.NotNil(job.UpdatedAt)
			s.Equal(job.Status, utils.Enum(statusTo))
			// Verify that the status is actually persisted in DB.
			s.Require().NoError(job.LoadFromDB(s.db))
			s.Equal(job.Status, utils.Enum(statusTo))
		}
	}
}

func (s *ModelsTestSuite) TestJob_StatusTransitions() {
	for status, name := range scheduler_proto.Job_Status_name {
		_, ok := models.JobStatusTransitions[scheduler_proto.Job_Status(status)]
		s.True(ok, "%s not found in models.JobStatusTransitions", name)
	}
}

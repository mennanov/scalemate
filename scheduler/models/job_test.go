package models_test

import (
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *ModelsTestSuite) TestJob_FromProto_ToProto() {
	now := time.Now().Unix()
	testCases := []struct {
		job  *scheduler_proto.Job
		mask string
	}{
		{
			job: &scheduler_proto.Job{
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
				RestartPolicy:  scheduler_proto.Job_RESTART_POLICY_ON_FAILURE,
				CpuLabels:      []string{"Intel Core i7 @ 2.20GHz", "Intel Core i5 @ 2.20GHz"},
				GpuLabels:      []string{"Intel Iris Pro 1536MB", "Intel Iris Pro 2000MB"},
				DiskLabels:     []string{"251GB APPLE SSD SM0256F"},
				MemoryLabels:   []string{"DDR3-1600MHz"},
				UsernameLabels: []string{"username1", "username2"},
				NameLabels:     []string{"node1", "node2"},
				OtherLabels:    []string{"Europe/Samara", "USA/SF"},
			},
			mask: "username,status,cpu_limit,cpu_class,memory_limit,gpu_limit,gpu_class,disk_limit," +
				"disk_class,run_config,created_at,updated_at,restart_policy,cpu_labels,gpu_labels,disk_labels," +
				"memory_labels,username_labels,name_labels,other_labels",
		},
		{
			job: &scheduler_proto.Job{
				Username: "username",
			},
			mask: "username",
		},
	}

	for _, testCase := range testCases {
		mask := fieldmask_utils.MaskFromString(testCase.mask)
		job := &models.Job{}
		err := job.FromProto(testCase.job)
		s.Require().NoError(err)
		// Create the job in DB.
		_, err = job.Create(s.db)
		s.Require().NoError(err)
		// Retrieve the same job from DB.
		jobFromDB := &models.Job{}
		s.db.First(jobFromDB, job.ID)
		p2, err := jobFromDB.ToProto(nil)
		s.Require().NoError(err)

		actual := &scheduler_proto.Job{}
		err = fieldmask_utils.StructToStruct(mask, p2, actual, generator.CamelCase, stringEye)
		s.Require().NoError(err)
		s.Equal(testCase.job, actual)
	}
}

func (s *ModelsTestSuite) TestJob_ScheduleForNode() {
	job := &models.Job{
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	node := &models.Node{
		Username:        "node_owner",
		Name:            "something",
		Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		CpuCapacity:     4,
		CpuAvailable:    2.5,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryCapacity:  8000,
		MemoryAvailable: 4000,
		GpuCapacity:     4,
		GpuAvailable:    2,
		GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
		DiskCapacity:    20000,
		DiskAvailable:   10000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		ConnectedAt:     now,
	}
	_, err = node.Create(s.db)
	s.Require().NoError(err)

	schedulingEvents, err := job.ScheduleForNode(s.db, node)
	s.Require().NoError(err)
	s.Equal(3, len(schedulingEvents))
	eventPayload, err := events.NewModelProtoFromEvent(schedulingEvents[0])
	s.Require().NoError(err)
	s.Equal(node.ID, eventPayload.(*scheduler_proto.Task).NodeId)
	nodeFromDB := &models.Node{Model: models.Model{ID: node.ID}}
	s.Require().NoError(nodeFromDB.LoadFromDB(s.db))
	s.Equal(float32(1), nodeFromDB.CpuAvailable)
	s.Equal(uint32(3000), nodeFromDB.MemoryAvailable)
	s.Equal(uint32(0), nodeFromDB.GpuAvailable)
	s.Equal(uint32(9000), nodeFromDB.DiskAvailable)
}

func (s *ModelsTestSuite) TestJob_SuitableNodeExistsSuccess() {
	job := &models.Job{
		Username:       "username",
		Status:         models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:       1.5,
		CpuClass:       models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:    1000,
		GpuLimit:       2,
		GpuClass:       models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:      1000,
		DiskClass:      models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy:  models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
		CpuLabels:      []string{"Intel Core i7 @ 2.20GHz", "Intel Core i5 @ 2.20GHz"},
		GpuLabels:      []string{"Intel Iris Pro 1536MB"},
		DiskLabels:     []string{"251GB APPLE SSD SM0256F"},
		MemoryLabels:   []string{"DDR3-1600MHz"},
		UsernameLabels: []string{"node_owner"},
		NameLabels:     []string{"node1", "node2"},
		OtherLabels:    []string{"sale20"},
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuModel:        "Intel Core i7 @ 2.20GHz",
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			MemoryModel:     "DDR3-1600MHz",
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			GpuModel:        "Intel Iris Pro 1536MB",
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskModel:       "251GB APPLE SSD SM0256F",
			Labels:          []string{"special_promo_label", "sale20"},
			ConnectedAt:     now,
		},
		// Offline node.
		{
			Username: "node_owner",
			Name:     "node2",
			Status:   models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	s.True(job.SuitableNodeExists(s.db))
}

func (s *ModelsTestSuite) TestJob_SuitableNodeExists_NotFoundByCpuClass() {
	job := &models.Job{
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	s.False(job.SuitableNodeExists(s.db))
}

func (s *ModelsTestSuite) TestJob_SuitableNodeExists_NotFoundByGpuClass() {
	job := &models.Job{
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	s.False(job.SuitableNodeExists(s.db))
}

func (s *ModelsTestSuite) TestJob_SuitableNodeExists_NotFoundByDiskClass() {
	job := &models.Job{
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	s.False(job.SuitableNodeExists(s.db))
}

func (s *ModelsTestSuite) TestJob_SuitableNodeExists_NotFoundByLabels() {
	job := &models.Job{
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
		CpuLabels:     []string{"Intel Core i5 @ 2.20GHz"},
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		{
			Username:     "node_owner",
			Name:         "node1",
			Status:       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:  4,
			CpuAvailable: 2.5,
			CpuClass:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:  models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			// "Intel Core i5 @ 2.20GHz" is requested.
			CpuModel:        "Intel Core i7 @ 2.20GHz",
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			GpuModel:        "Intel Iris Pro 1536MB",
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskModel:       "251GB APPLE SSD SM0256F",
			ConnectedAt:     now,
		},
	}
	for _, node := range nodes {
		_, err = node.Create(s.db)
		s.Require().NoError(err)
	}

	s.False(job.SuitableNodeExists(s.db))
}

func (s *ModelsTestSuite) TestJob_FindSuitableNode_OneAvailable() {
	job := &models.Job{
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
		// Offline node.
		{
			Username: "node_owner",
			Name:     "node2",
			Status:   models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
		// Does not satisfy by GpuClass.
		{
			Username:        "node_owner",
			Name:            "node3",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 7000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
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
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    3,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
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
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
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
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 4000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   15000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
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
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    3,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    2,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
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
		Username:      "username",
		Status:        models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:      1.5,
		CpuClass:      models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
		DiskLimit:     1000,
		DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)

	now := time.Now()

	nodes := []*models.Node{
		// This node satisfies criteria.
		{
			Username:        "node_owner",
			Name:            "node1",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    3,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now,
			ScheduledAt:     now,
		},
		// This node satisfies criteria and is the least loaded (CpuAvailable).
		{
			Username:        "node_owner",
			Name:            "node2",
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
			CpuCapacity:     4,
			CpuAvailable:    2.5,
			CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			MemoryCapacity:  8000,
			MemoryAvailable: 6000,
			GpuCapacity:     4,
			GpuAvailable:    3,
			GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
			DiskCapacity:    20000,
			DiskAvailable:   10000,
			DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			ConnectedAt:     now.Add(-time.Minute * 2),
			ScheduledAt:     now.Add(-time.Minute),
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

func (s *ModelsTestSuite) TestJob_UpdateStatusForNodeFailedTasks() {
	jobRescheduleOnNodeFailure := &models.Job{
		Status:        models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err := jobRescheduleOnNodeFailure.Create(s.db)
	s.Require().NoError(err)

	jobRestartOnFailure := &models.Job{
		Status:        models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_ON_FAILURE),
	}
	_, err = jobRestartOnFailure.Create(s.db)
	s.Require().NoError(err)

	jobNotListed := &models.Job{
		Status:        models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		RestartPolicy: models.Enum(scheduler_proto.Job_RESTART_POLICY_RESCHEDULE_ON_NODE_FAILURE),
	}
	_, err = jobNotListed.Create(s.db)
	s.Require().NoError(err)

	var jobs models.Jobs
	jobsEvents, err := jobs.UpdateStatusForNodeFailedTasks(
		s.db, []uint64{jobRescheduleOnNodeFailure.ID, jobRestartOnFailure.ID})
	s.Require().NoError(err)
	s.Len(jobsEvents, 2)
	s.Require().NoError(jobRescheduleOnNodeFailure.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_PENDING), jobRescheduleOnNodeFailure.Status)
	s.Require().NoError(jobRestartOnFailure.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_FINISHED), jobRestartOnFailure.Status)
	s.Require().NoError(jobNotListed.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_SCHEDULED), jobNotListed.Status)
}

func (s *ModelsTestSuite) TestJobs_FindPendingForNode() {
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
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	jobs := []*models.Job{
		// This Job satisfies criteria.
		{
			Username:       "job1_username",
			Status:         models.Enum(scheduler_proto.Job_STATUS_PENDING),
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
			CpuLimit:    3.5,
			MemoryLimit: 2000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username:    "job4_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			// CpuLimit is too high.
			CpuLimit:    4,
			MemoryLimit: 2000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username:    "job5_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    3,
			// Memory limit is too high.
			MemoryLimit: 8000,
			GpuLimit:    2,
			DiskLimit:   2000,
		},
		{
			Username:    "job6_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    3,
			MemoryLimit: 4000,
			// GpuLimit is too high.
			GpuLimit:  6,
			DiskLimit: 2000,
		},
		{
			Username:    "job7_username",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
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
	pendingJobsUsernames := make(map[string]struct{}, 0)
	for _, job := range pendingJobs {
		pendingJobsUsernames[job.Username] = struct{}{}
	}
	s.Equal(map[string]struct{}{"job1_username": {}, "job2_username": {}, "job3_username": {}}, pendingJobsUsernames)
}

func stringEye(s string) string {
	return s
}

package models_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ModelsTestSuite) TestNode_Create() {
	for _, node := range []*models.Node{
		{
			Node: scheduler_proto.Node{
				Id:       1,
				Username: "username",
				Name:     "name",
			},
		},
		new(models.Node),
		{
			Node: scheduler_proto.Node{
				Username: "username",
				Name:     "another name",
				Labels:   []string{"label1", "label2"},
			},
		},
	} {
		_, err := node.Create(s.db)
		s.Require().NoError(err)
		s.NotNil(node.Id)
		s.False(node.CreatedAt.IsZero())
		s.Nil(node.UpdatedAt)
		nodeFromDB, err := models.NewNodeFromDB(s.db, node.Id)
		s.Require().NoError(err)
		s.Equal(node, nodeFromDB)
	}
}

func (s *ModelsTestSuite) TestNode_AllocateResources() {
	node := models.NewNodeFromProto(&scheduler_proto.Node{
		CpuAvailable:    4,
		GpuAvailable:    4,
		MemoryAvailable: 4,
		DiskAvailable:   4,
	})

	for _, testCase := range []struct {
		newRequest     *models.ResourceRequest
		currentRequest *models.ResourceRequest
		expectedOutput map[string]interface{}
	}{
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    3,
				Memory: 3,
				Disk:   3,
				Gpu:    3,
			}),
			currentRequest: nil,
			expectedOutput: map[string]interface{}{
				"cpu_available":    uint32(1),
				"gpu_available":    uint32(1),
				"memory_available": uint32(1),
				"disk_available":   uint32(1),
			},
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    4,
				Memory: 4,
				Disk:   4,
				Gpu:    4,
			}),
			currentRequest: nil,
			expectedOutput: map[string]interface{}{
				"cpu_available":    uint32(0),
				"gpu_available":    uint32(0),
				"memory_available": uint32(0),
				"disk_available":   uint32(0),
			},
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    5, // Overflows available CPU.
				Memory: 4,
				Disk:   4,
				Gpu:    4,
			}),
			currentRequest: nil,
			expectedOutput: nil,
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    4,
				Memory: 5, // Overflows available Memory.
				Disk:   4,
				Gpu:    4,
			}),
			currentRequest: nil,
			expectedOutput: nil,
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    4,
				Memory: 4,
				Disk:   5, // Overflows available Disk.
				Gpu:    4,
			}),
			currentRequest: nil,
			expectedOutput: nil,
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    4,
				Memory: 4,
				Disk:   4,
				Gpu:    5, // Overflows available GPU.
			}),
			currentRequest: nil,
			expectedOutput: nil,
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    8,
				Memory: 8,
				Disk:   8,
				Gpu:    8,
			}),
			currentRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    4,
				Memory: 4,
				Disk:   4,
				Gpu:    4,
			}),
			expectedOutput: map[string]interface{}{
				"cpu_available":    uint32(0),
				"gpu_available":    uint32(0),
				"memory_available": uint32(0),
				"disk_available":   uint32(0),
			},
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    9, // Overflows available CPU.
				Memory: 8,
				Disk:   8,
				Gpu:    8,
			}),
			currentRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    4,
				Memory: 4,
				Disk:   4,
				Gpu:    4,
			}),
			expectedOutput: nil,
		},
		{
			newRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    2,
				Memory: 2,
				Disk:   2,
				Gpu:    2,
			}),
			currentRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    4,
				Memory: 4,
				Disk:   4,
				Gpu:    4,
			}),
			expectedOutput: map[string]interface{}{
				"cpu_available":    uint32(6),
				"gpu_available":    uint32(6),
				"memory_available": uint32(6),
				"disk_available":   uint32(6),
			},
		},
	} {
		output, err := node.AllocateResources(testCase.newRequest, testCase.currentRequest)
		if testCase.expectedOutput != nil {
			s.Require().NoError(err)
			s.Equal(testCase.expectedOutput, output)
		} else {
			s.Error(err)
			s.Nil(output)
		}
	}
}

//func (s *ModelsTestSuite) TestFindNodesForContainer() {
//	nodes := []*models.Node{
//		{
//			Node: scheduler_proto.Node{
//				Id:                     1,
//				CpuClass:               scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
//				CpuAvailable:           8,
//				GpuClass:               scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
//				GpuAvailable:           4,
//				DiskClass:              scheduler_proto.DiskClass_DISK_CLASS_SSD,
//				DiskAvailable:          10000,
//				MemoryAvailable:        4000,
//				Username:               "username",
//				Name:                   "node1",
//				NetworkIngressCapacity: 50,
//				NetworkEgressCapacity:  50,
//				Status:                 scheduler_proto.Node_ONLINE,
//				Labels:                 []string{"Intel i7", "Nvidia P100"},
//			},
//		},
//		{
//			Node: scheduler_proto.Node{
//				Id:              2,
//				CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
//				CpuAvailable:    4,
//				GpuClass:        scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
//				GpuAvailable:    2,
//				DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_HDD,
//				DiskAvailable:   5000,
//				MemoryAvailable: 2000,
//				Username:        "username",
//				Name:            "node2",
//				Status:          scheduler_proto.Node_ONLINE,
//				Labels:          []string{"AMD Ryzen", "Nvidia RTX 2060"},
//			},
//		},
//		{
//			Node: scheduler_proto.Node{
//				Id:              3,
//				CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE,
//				CpuAvailable:    8,
//				GpuClass:        scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE,
//				GpuAvailable:    4,
//				DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_HDD,
//				DiskAvailable:   10000,
//				MemoryAvailable: 4000,
//				Username:        "username",
//				Name:            "node3",
//				Status:          scheduler_proto.Node_ONLINE,
//				Labels:          []string{"Intel i3", "Nvidia P80", "Intel other"},
//			},
//		},
//	}
//	for _, n := range nodes {
//		_, err := n.Create(s.db)
//		s.Require().NoError(err)
//	}
//	for _, testCase := range []struct {
//		container       *models.Container
//		resourceRequest *models.ResourceRequest
//		expectedNodeIds []int64
//	}{
//		// Labels focused test cases:
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				Labels: []string{"intel", "nvidia"},
//			}),
//			expectedNodeIds: []int64{1, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				Labels: []string{"amd", "nvidia"},
//			}),
//			expectedNodeIds: []int64{2},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				Labels: []string{"amd"},
//			}),
//			expectedNodeIds: []int64{2},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				Labels: []string{"nvidia"},
//			}),
//			expectedNodeIds: []int64{1, 2, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				Labels: []string{"amd", "nvidia", "intel"},
//			}),
//			expectedNodeIds: []int64{},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				Labels: []string{"other label", "nvidia", "something else"},
//			}),
//			expectedNodeIds: []int64{},
//		},
//		{
//			container:       models.NewContainerFromProto(&scheduler_proto.Container{}),
//			expectedNodeIds: []int64{1, 2, 3},
//		},
//		// Classes focused test cases:
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				CpuClassMin:  scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
//				CpuClassMax:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
//				GpuClassMin:  scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
//				GpuClassMax:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
//				DiskClassMin: scheduler_proto.DiskClass_DISK_CLASS_HDD,
//				DiskClassMax: scheduler_proto.DiskClass_DISK_CLASS_SSD,
//			}),
//			expectedNodeIds: []int64{1, 2, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				CpuClassMin:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
//				CpuClassMax:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
//				GpuClassMin:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
//				GpuClassMax:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
//				DiskClassMin: scheduler_proto.DiskClass_DISK_CLASS_SSD,
//				DiskClassMax: scheduler_proto.DiskClass_DISK_CLASS_SSD,
//			}),
//			expectedNodeIds: []int64{1},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				CpuClassMin:  scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE,
//				CpuClassMax:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
//				GpuClassMin:  scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE,
//				GpuClassMax:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
//				DiskClassMin: scheduler_proto.DiskClass_DISK_CLASS_HDD,
//				DiskClassMax: scheduler_proto.DiskClass_DISK_CLASS_SSD,
//			}),
//			expectedNodeIds: []int64{1, 3},
//		},
//		// Resources focused test cases:
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Cpu:    1,
//				Memory: 1,
//				Disk:   1,
//				Gpu:    1,
//			}),
//			expectedNodeIds: []int64{1, 2, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Cpu:    8,
//				Gpu:    4,
//				Memory: 4000,
//				Disk:   10000,
//			}),
//			expectedNodeIds: []int64{1, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Cpu: 100,
//			}),
//			expectedNodeIds: []int64{},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Cpu: 8,
//			}),
//			expectedNodeIds: []int64{1, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Gpu: 2,
//			}),
//			expectedNodeIds: []int64{1, 2, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Memory: 2000,
//			}),
//			expectedNodeIds: []int64{1, 2, 3},
//		},
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Disk: 5000,
//			}),
//			expectedNodeIds: []int64{1, 2, 3},
//		},
//		// All in one:
//		{
//			container: models.NewContainerFromProto(&scheduler_proto.Container{
//				Labels:            []string{"intel", "nvidia"},
//				CpuClassMin:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
//				CpuClassMax:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
//				GpuClassMin:       scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
//				GpuClassMax:       scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
//				DiskClassMin:      scheduler_proto.DiskClass_DISK_CLASS_SSD,
//				DiskClassMax:      scheduler_proto.DiskClass_DISK_CLASS_SSD,
//				NetworkIngressMin: 50,
//				NetworkEgressMin:  50,
//			}),
//			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
//				Cpu:    8,
//				Gpu:    4,
//				Disk:   10000,
//				Memory: 4000,
//			}),
//			expectedNodeIds: []int64{1},
//		},
//	} {
//		_, err := testCase.container.Create(s.db)
//		s.Require().NoError(err)
//		foundNodes, err := models.NodesForContainer(testCase.container, testCase.resourceRequest)
//		s.Require().NoError(err)
//		actualNodeIds := make(map[int64]struct{})
//		for _, node := range foundNodes {
//			actualNodeIds[node.Id] = struct{}{}
//		}
//		expectedNodeIds := make(map[int64]struct{})
//		for _, nodeId := range testCase.expectedNodeIds {
//			expectedNodeIds[nodeId] = struct{}{}
//		}
//		s.Equal(expectedNodeIds, actualNodeIds)
//	}
//
//}

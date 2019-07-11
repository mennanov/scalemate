package models_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ModelsTestSuite) TestContainer_Create() {
	s.Run("empty container", func() {
		s.T().Parallel()
		container := &models.Container{}
		s.Require().NoError(container.Create(s.db))
		s.NotNil(container.Id)
		s.False(container.CreatedAt.IsZero())
		s.Nil(container.UpdatedAt)
	})

	s.Run("full container with node and labels", func() {
		s.T().Parallel()
		node := models.Node{}
		err := node.Create(s.db)
		s.Require().NoError(err)

		container := models.NewContainerFromProto(&scheduler_proto.Container{
			Id:                42,
			NodeId:            &node.Id,
			NodePricingId:     nil,
			Username:          "username",
			Status:            scheduler_proto.Container_RUNNING,
			StatusMessage:     "status message",
			Image:             "image",
			NetworkIngressMin: 1,
			NetworkEgressMin:  1,
			CpuClassMin:       scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
			CpuClassMax:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
			GpuClassMin:       scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
			GpuClassMax:       scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
			DiskClassMin:      scheduler_proto.DiskClass_DISK_CLASS_HDD,
			DiskClassMax:      scheduler_proto.DiskClass_DISK_CLASS_SSD,
			Labels:            []string{"label1", "label2"},
			NodeAuthToken:     []byte("token"),
			SchedulingStrategy: []*scheduler_proto.SchedulingStrategy{
				{
					Strategy:             scheduler_proto.SchedulingStrategy_CHEAPEST,
					DifferencePercentage: 15,
				},
				{
					Strategy:             scheduler_proto.SchedulingStrategy_MOST_RELIABLE,
					DifferencePercentage: 20,
				},
			},
			MaxPriceLimit: 10,
		})
		s.Require().NoError(container.Create(s.db))
		s.NotNil(container.Id)
		s.False(container.CreatedAt.IsZero())
		s.Nil(container.UpdatedAt)
		// Get the same container from DB and verify they are equal.
		containerFromDB, err := models.NewContainerFromDB(s.db, container.Id)
		s.Require().NoError(err)
		s.Equal(container, containerFromDB)
	})
}

func (s *ModelsTestSuite) TestContainer_ValidateNewStatus() {
	for statusFrom, statusesTo := range models.ContainerStatusTransitions {
		for _, statusTo := range statusesTo {
			container := &models.Container{Container: scheduler_proto.Container{Status: statusFrom}}
			err := container.Create(s.db)
			s.Require().NoError(err)
			s.Nil(container.UpdatedAt)
			s.Require().NoError(container.ValidateNewStatus(statusTo))
		}
	}
}

func (s *ModelsTestSuite) TestContainer_StatusTransitions() {
	for status, name := range scheduler_proto.Container_Status_name {
		_, ok := models.ContainerStatusTransitions[scheduler_proto.Container_Status(status)]
		s.True(ok, "%s not found in models.ContainerStatusTransitions", name)
	}
}

func (s *ModelsTestSuite) TestListContainers() {
	now := time.Now()
	minuteLater := now.Add(time.Minute)
	minuteEarlier := now.Add(-time.Minute)
	containers := []*models.Container{
		{
			Container: scheduler_proto.Container{
				Id:        1,
				Username:  "user1",
				Status:    scheduler_proto.Container_RUNNING,
				CreatedAt: minuteEarlier,
				UpdatedAt: &now,
			},
		},
		{
			Container: scheduler_proto.Container{
				Id:        2,
				Username:  "user1",
				Status:    scheduler_proto.Container_STOPPED,
				CreatedAt: now,
				UpdatedAt: &minuteLater,
			},
		},
		{
			Container: scheduler_proto.Container{
				Id:       3,
				Username: "user2",
				Status:   scheduler_proto.Container_RUNNING,
			},
		},
	}

	for _, container := range containers {
		err := container.Create(s.db)
		s.Require().NoError(err)
	}

	s.Run("containers found", func() {
		for _, testCase := range []struct {
			username             string
			request              *scheduler_proto.ListContainersRequest
			expectedContainerIds []int64
			expectedCount        uint32
		}{
			{
				username: "user1",
				request: &scheduler_proto.ListContainersRequest{
					Limit: 10,
				},
				expectedContainerIds: []int64{2, 1},
				expectedCount:        2,
			},
			{
				username: "user1",
				request: &scheduler_proto.ListContainersRequest{
					Limit:    10,
					Ordering: scheduler_proto.ListContainersRequest_CREATED_AT_ASC,
				},
				expectedContainerIds: []int64{1, 2},
				expectedCount:        2,
			},
			{
				username: "user2",
				request: &scheduler_proto.ListContainersRequest{
					Limit: 10,
				},
				expectedContainerIds: []int64{3},
				expectedCount:        1,
			},
			{
				username: "user1",
				request: &scheduler_proto.ListContainersRequest{
					Limit:  1,
					Offset: 1,
				},
				expectedContainerIds: []int64{1},
				expectedCount:        2,
			},
			{
				username: "user1",
				request: &scheduler_proto.ListContainersRequest{
					Limit:  10,
					Status: []scheduler_proto.Container_Status{scheduler_proto.Container_RUNNING},
				},
				expectedContainerIds: []int64{1},
				expectedCount:        1,
			},
			{
				username: "user1",
				request: &scheduler_proto.ListContainersRequest{
					Limit: 10,
					Status: []scheduler_proto.Container_Status{
						scheduler_proto.Container_RUNNING,
						scheduler_proto.Container_STOPPED,
					},
					Ordering: scheduler_proto.ListContainersRequest_CREATED_AT_ASC,
				},
				expectedContainerIds: []int64{1, 2},
				expectedCount:        2,
			},
		} {
			actualContainers, count, err := models.ListContainers(s.db, testCase.username, testCase.request)
			s.Require().NoError(err)
			s.EqualValues(count, testCase.expectedCount)
			ids := make([]int64, len(actualContainers))
			for i, container := range actualContainers {
				ids[i] = container.Id
			}
			s.Equal(testCase.expectedContainerIds, ids)
		}
	})

	s.Run("containers not found", func() {
		s.T().Parallel()
		var missingStatuses []scheduler_proto.Container_Status
		existingStatuses := make(map[scheduler_proto.Container_Status]struct{})
		for _, container := range containers {
			existingStatuses[container.Status] = struct{}{}
		}

		for status := range scheduler_proto.Container_Status_name {
			s := scheduler_proto.Container_Status(status)
			_, ok := existingStatuses[s]
			if !ok {
				missingStatuses = append(missingStatuses, s)
			}
		}
		for _, testCase := range []struct {
			username string
			request  *scheduler_proto.ListContainersRequest
		}{
			{
				username: "user0",
				request:  &scheduler_proto.ListContainersRequest{},
			},
			{
				username: "user1",
				request: &scheduler_proto.ListContainersRequest{
					Status: missingStatuses,
				},
			},
		} {
			containers, count, err := models.ListContainers(s.db, testCase.username, testCase.request)
			testutils.AssertErrorCode(s.T(), err, codes.NotFound)
			s.Equal(uint32(0), count)
			s.Nil(containers)
		}
	})
}

func (s *ModelsTestSuite) TestContainer_NodesForScheduling() {
	for _, testData := range []struct {
		node       *models.Node
		pricings   []*models.NodePricing
		containers []*models.Container
	}{
		{
			&models.Node{
				Node: scheduler_proto.Node{
					Id:                     1,
					CpuClass:               scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
					CpuAvailable:           8,
					GpuClass:               scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
					GpuAvailable:           4,
					DiskClass:              scheduler_proto.DiskClass_DISK_CLASS_SSD,
					DiskAvailable:          10000,
					MemoryAvailable:        4000,
					Username:               "username",
					Name:                   "node1",
					NetworkIngressCapacity: 50,
					NetworkEgressCapacity:  50,
					Status:                 scheduler_proto.Node_ONLINE,
					Labels:                 []string{"Intel i7", "Nvidia P100"},
				},
			},
			[]*models.NodePricing{
				// This pricing is never expected to be used as it is considered obsolete.
				{
					NodePricing: scheduler_proto.NodePricing{
						Id:          1,
						CpuPrice:    1,
						MemoryPrice: 1,
						GpuPrice:    1,
						DiskPrice:   1,
					},
				},
				{
					NodePricing: scheduler_proto.NodePricing{
						Id:          2,
						CpuPrice:    2,
						MemoryPrice: 2,
						GpuPrice:    2,
						DiskPrice:   2,
					},
				},
			},
			[]*models.Container{
				{
					Container: scheduler_proto.Container{
						Status: scheduler_proto.Container_NODE_FAILED,
					},
				},
				{
					Container: scheduler_proto.Container{
						Status: scheduler_proto.Container_RUNNING,
					},
				},
			},
		},
		{
			&models.Node{
				Node: scheduler_proto.Node{
					Id:              2,
					CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
					CpuAvailable:    4,
					GpuClass:        scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
					GpuAvailable:    2,
					DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_HDD,
					DiskAvailable:   5000,
					MemoryAvailable: 2000,
					Username:        "username",
					Name:            "node2",
					Status:          scheduler_proto.Node_ONLINE,
					Labels:          []string{"AMD Ryzen", "Nvidia RTX 2060"},
				},
			},
			[]*models.NodePricing{
				{
					NodePricing: scheduler_proto.NodePricing{
						Id:          3,
						CpuPrice:    1,
						MemoryPrice: 1,
						GpuPrice:    1,
						DiskPrice:   3,
					},
				},
			},
			[]*models.Container{
				{
					Container: scheduler_proto.Container{
						Status: scheduler_proto.Container_SCHEDULED,
					},
				},
				{
					Container: scheduler_proto.Container{
						Status: scheduler_proto.Container_RUNNING,
					},
				},
				{
					Container: scheduler_proto.Container{
						Status: scheduler_proto.Container_EVICTED,
					},
				},
			},
		},
		{
			&models.Node{
				Node: scheduler_proto.Node{
					Id:              3,
					CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE,
					CpuAvailable:    8,
					GpuClass:        scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE,
					GpuAvailable:    4,
					DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_HDD,
					DiskAvailable:   10000,
					MemoryAvailable: 4000,
					Username:        "username",
					Name:            "node3",
					Status:          scheduler_proto.Node_ONLINE,
					Labels:          []string{"Intel i3", "Nvidia P80", "Intel other"},
				},
			},
			[]*models.NodePricing{
				{
					NodePricing: scheduler_proto.NodePricing{
						Id:          4,
						CpuPrice:    3,
						MemoryPrice: 3,
						GpuPrice:    3,
						DiskPrice:   4,
					},
				},
			},
			nil,
		},
		{
			// Node with no GPU.
			&models.Node{
				Node: scheduler_proto.Node{
					Id:              4,
					CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE,
					CpuAvailable:    8,
					DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_HDD,
					DiskAvailable:   10000,
					MemoryAvailable: 4000,
					Username:        "username",
					Name:            "node4",
					Status:          scheduler_proto.Node_ONLINE,
					Labels:          []string{"Intel i3", "Nvidia P80", "Intel other"},
				},
			},
			[]*models.NodePricing{
				{
					NodePricing: scheduler_proto.NodePricing{
						Id:          5,
						CpuPrice:    3,
						MemoryPrice: 3,
						GpuPrice:    3,
						DiskPrice:   4,
					},
				},
			},
			[]*models.Container{
				{
					Container: scheduler_proto.Container{
						Status: scheduler_proto.Container_NEW,
					},
				},
				{
					Container: scheduler_proto.Container{
						Status: scheduler_proto.Container_FAILED,
					},
				},
			},
		},
		// Offline Node.
		{
			&models.Node{
				Node: scheduler_proto.Node{
					Id:              5,
					CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE,
					CpuAvailable:    8,
					GpuClass:        scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE,
					GpuAvailable:    8,
					DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_HDD,
					DiskAvailable:   100000,
					MemoryAvailable: 40000,
					Username:        "username",
					Name:            "node5",
					Status:          scheduler_proto.Node_OFFLINE,
					Labels:          []string{"Intel i3", "Nvidia P80", "Intel other"},
				},
			},
			[]*models.NodePricing{
				{
					NodePricing: scheduler_proto.NodePricing{
						Id:          6,
						CpuPrice:    1,
						MemoryPrice: 1,
						GpuPrice:    1,
						DiskPrice:   1,
					},
				},
			},
			nil,
		},
	} {
		err := testData.node.Create(s.db)
		s.Require().NoError(err)
		for _, pricing := range testData.pricings {
			pricing.NodeId = testData.node.Id
			s.Require().NoError(pricing.Create(s.db))
		}
		for _, container := range testData.containers {
			container.NodeId = &testData.node.Id
			s.Require().NoError(container.Create(s.db))
		}
	}
	emptyResourceRequest := new(models.ResourceRequest)
	for _, testCase := range []struct {
		container       *models.Container
		resourceRequest *models.ResourceRequest
		expectedNodeIds map[int64]bool
	}{
		// Labels focused test cases:
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				Labels: []string{"intel", "nvidia"},
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{1: true, 3: true, 4: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				Labels: []string{"amd", "nvidia"},
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{2: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				Labels: []string{"amd"},
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{2: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				Labels: []string{"nvidia"},
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{1: true, 2: true, 3: true, 4: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				Labels: []string{"amd", "nvidia", "intel"},
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: nil,
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				Labels: []string{"other label", "nvidia", "something else"},
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: nil,
		},
		{
			container:       models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: emptyResourceRequest,
			// Container with no constraints should match all the Nodes.
			expectedNodeIds: map[int64]bool{1: true, 2: true, 3: true, 4: true},
		},
		// Classes focused test cases:
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				CpuClassMin:  scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
				CpuClassMax:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				DiskClassMin: scheduler_proto.DiskClass_DISK_CLASS_HDD,
				DiskClassMax: scheduler_proto.DiskClass_DISK_CLASS_SSD,
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{1: true, 2: true, 3: true, 4: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				CpuClassMin:  scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
				CpuClassMax:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				GpuClassMin:  scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
				GpuClassMax:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				DiskClassMin: scheduler_proto.DiskClass_DISK_CLASS_HDD,
				DiskClassMax: scheduler_proto.DiskClass_DISK_CLASS_SSD,
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{1: true, 2: true, 3: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				CpuClassMin:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				CpuClassMax:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				GpuClassMin:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				GpuClassMax:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				DiskClassMin: scheduler_proto.DiskClass_DISK_CLASS_SSD,
				DiskClassMax: scheduler_proto.DiskClass_DISK_CLASS_SSD,
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{1: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				CpuClassMin:  scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE,
				CpuClassMax:  scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				GpuClassMin:  scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE,
				GpuClassMax:  scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				DiskClassMin: scheduler_proto.DiskClass_DISK_CLASS_HDD,
				DiskClassMax: scheduler_proto.DiskClass_DISK_CLASS_SSD,
			}),
			resourceRequest: emptyResourceRequest,
			expectedNodeIds: map[int64]bool{1: true, 3: true},
		},
		// Resources focused test cases:
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    1,
				Memory: 1,
				Disk:   1,
			}),
			expectedNodeIds: map[int64]bool{1: true, 2: true, 3: true, 4: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    1,
				Memory: 1,
				Disk:   1,
				Gpu:    1,
			}),
			expectedNodeIds: map[int64]bool{1: true, 2: true, 3: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    8,
				Gpu:    4,
				Memory: 4000,
				Disk:   10000,
			}),
			expectedNodeIds: map[int64]bool{1: true, 3: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu: 100,
			}),
			expectedNodeIds: nil,
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu: 8,
			}),
			expectedNodeIds: map[int64]bool{1: true, 3: true, 4: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Gpu: 3,
			}),
			expectedNodeIds: map[int64]bool{1: true, 3: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Memory: 4000,
			}),
			expectedNodeIds: map[int64]bool{1: true, 3: true, 4: true},
		},
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Disk: 10000,
			}),
			expectedNodeIds: map[int64]bool{1: true, 3: true, 4: true},
		},
		// All in one:
		{
			container: models.NewContainerFromProto(&scheduler_proto.Container{
				Labels:            []string{"intel", "nvidia"},
				CpuClassMin:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				CpuClassMax:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				GpuClassMin:       scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				GpuClassMax:       scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				DiskClassMin:      scheduler_proto.DiskClass_DISK_CLASS_SSD,
				DiskClassMax:      scheduler_proto.DiskClass_DISK_CLASS_SSD,
				NetworkIngressMin: 50,
				NetworkEgressMin:  50,
				MaxPriceLimit:     8*2 + 4*2 + 10000*2 + 4000*2,
			}),
			resourceRequest: models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				Cpu:    8,
				Gpu:    4,
				Disk:   10000,
				Memory: 4000,
			}),
			expectedNodeIds: map[int64]bool{1: true},
		},
	} {
		err := testCase.container.Create(s.db)
		s.Require().NoError(err)
		nodesFound, err := testCase.container.NodesForScheduling(s.db, testCase.resourceRequest)
		s.Require().NoError(err)
		if len(testCase.expectedNodeIds) != 0 {
			actualNodeIds := make(map[int64]bool)
			for _, n := range nodesFound {
				// Check there are no duplicates.
				s.Require().False(actualNodeIds[n.Id])
				actualNodeIds[n.Id] = true
			}
			s.Equal(testCase.expectedNodeIds, actualNodeIds)
		} else {
			s.Len(nodesFound, 0)
		}
	}
}

func (s *ModelsTestSuite) TestContainer_ResourceRequest() {
	container := new(models.Container)
	s.Require().NoError(container.Create(s.db))

	s.Run("not found", func() {
		request, err := container.CurrentResourceRequest(s.db)
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(request)
	})

	s.Run("returns the most recent CurrentResourceRequest", func() {
		for i := 0; i < 4; i++ {
			recentRequest := models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				ContainerId: container.Id,
				Status:      scheduler_proto.ResourceRequest_CONFIRMED,
				CreatedAt:   time.Unix(int64(i), 0),
			})
			s.Require().NoError(recentRequest.Create(s.db))
			request, err := container.CurrentResourceRequest(s.db)
			s.Require().NoError(err)
			s.Equal(recentRequest, request)
		}
	})
}

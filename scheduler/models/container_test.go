package models_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ModelsTestSuite) TestContainer_Create() {
	s.Run("empty container", func() {
		s.T().Parallel()
		container := &models.Container{}
		event, err := container.Create(s.db)
		s.Require().NoError(err)
		s.NotNil(event)
		s.NotNil(container.Id)
		s.False(container.CreatedAt.IsZero())
		s.Nil(container.UpdatedAt)
	})

	s.Run("full container with node and labels", func() {
		s.T().Parallel()
		node := models.Node{}
		_, err := node.Create(s.db)
		s.Require().NoError(err)

		container := models.NewContainerFromProto(&scheduler_proto.Container{
			NodeId:            &node.Id,
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
			AgentAuthToken:    []byte("token"),
		})
		event, err := container.Create(s.db)
		s.Require().NoError(err)
		s.NotNil(event)
		s.NotNil(container.Id)
		s.False(container.CreatedAt.IsZero())
		s.Nil(container.UpdatedAt)
		// Get the same container from DB and verify they are equal.
		containerFromDB, err := models.NewContainerFromDB(s.db, container.Id, false, s.logger)
		s.Require().NoError(err)
		s.Equal(container, containerFromDB)
	})
}

func (s *ModelsTestSuite) TestContainer_UpdateStatus() {
	for statusFrom, statusesTo := range models.ContainerStatusTransitions {
		for _, statusTo := range statusesTo {
			container := &models.Container{Container: scheduler_proto.Container{Status: statusFrom}}
			_, err := container.Create(s.db)
			s.Require().NoError(err)
			s.Nil(container.UpdatedAt)

			_, err = container.UpdateStatus(s.db, statusTo, s.logger)
			s.Require().NoError(err)
			s.NotNil(container.UpdatedAt)
			s.Equal(statusTo, container.Status)
			// Verify the status is updated in DB.
			containerFromDB, err := models.NewContainerFromDB(s.db, container.Id, false, s.logger)
			s.Require().NoError(err)
			s.Equal(container.Status, containerFromDB.Status)
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
		_, err := container.Create(s.db)
		s.Require().NoError(err)
	}

	for _, testCase := range []struct {
		request              *scheduler_proto.ListContainersRequest
		expectedContainerIds []int64
		expectedCount        uint32
	}{
		{
			request: &scheduler_proto.ListContainersRequest{
				Username: "user1",
				Limit:    10,
			},
			expectedContainerIds: []int64{2, 1},
			expectedCount:        2,
		},
		{
			request: &scheduler_proto.ListContainersRequest{
				Username: "user1",
				Limit:    10,
				Ordering: scheduler_proto.ListContainersRequest_CREATED_AT_ASC,
			},
			expectedContainerIds: []int64{1, 2},
			expectedCount:        2,
		},
		{
			request: &scheduler_proto.ListContainersRequest{
				Username: "user2",
				Limit:    10,
			},
			expectedContainerIds: []int64{3},
			expectedCount:        1,
		},
		{
			request: &scheduler_proto.ListContainersRequest{
				Username: "user1",
				Limit:    1,
				Offset:   1,
			},
			expectedContainerIds: []int64{1},
			expectedCount:        2,
		},
		{
			request: &scheduler_proto.ListContainersRequest{
				Username: "user1",
				Limit:    10,
				Status:   []scheduler_proto.Container_Status{scheduler_proto.Container_RUNNING},
			},
			expectedContainerIds: []int64{1},
			expectedCount:        1,
		},
		{
			request: &scheduler_proto.ListContainersRequest{
				Username: "user1",
				Limit:    10,
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
		actualContainers, count, err := models.ListContainers(s.db, testCase.request)
		s.Require().NoError(err)
		s.EqualValues(count, testCase.expectedCount)
		ids := make([]int64, len(actualContainers))
		for i, container := range actualContainers {
			ids[i] = container.Id
		}
		s.Equal(testCase.expectedContainerIds, ids)
	}
}

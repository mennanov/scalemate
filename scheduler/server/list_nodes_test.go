package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestListNodes() {
	ctx := context.Background()

	minuteAgo := time.Now().Add(-time.Minute)
	hourAgo := time.Now().Add(-time.Minute * 60)
	dayAgo := time.Now().Add(-time.Hour * 24)

	for _, testCase := range []struct {
		nodes              []*models.Node
		nodeNamesExpected  []string
		totalCountExpected uint32
		request            *scheduler_proto.ListNodesRequest
	}{
		{
			nodes: []*models.Node{
				{
					Name:        "node1",
					Username:    "username1",
					ConnectedAt: &minuteAgo,
				},
				{
					Name:        "node2",
					Username:    "username1",
					ConnectedAt: &hourAgo,
				},
				{
					Name:     "node3",
					Username: "username2",
				},
			},
			nodeNamesExpected:  []string{"node1"},
			totalCountExpected: 2,
			request:            &scheduler_proto.ListNodesRequest{UsernameLabels: []string{"username1"}, Limit: 1},
		},
		{
			nodes: []*models.Node{
				{
					Name:            "node1",
					Username:        "username1",
					ConnectedAt:     &minuteAgo,
					Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
					CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
					CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE),
					CpuAvailable:    4,
					CpuModel:        "Intel i7",
					MemoryAvailable: 2000,
					MemoryModel:     "DDR3",
					GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ADVANCED),
					GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
					GpuAvailable:    2,
					GpuModel:        "Nvidia",
					DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
					DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
					DiskAvailable:   10000,
					DiskModel:       "Seagate",
					Labels:          []string{"label1", "label2", "label3"},
					TasksFinished:   100,
					TasksFailed:     50,
				},
				{
					Name:        "node2",
					Username:    "username1",
					ConnectedAt: &hourAgo,
					Status:      models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
				},
				{
					Name:     "node3",
					Username: "username1",
					Status:   models.Enum(scheduler_proto.Node_STATUS_SHUTTING_DOWN),
				},
			},
			nodeNamesExpected:  []string{"node1"},
			totalCountExpected: 1,
			request: &scheduler_proto.ListNodesRequest{
				Status: []scheduler_proto.Node_Status{
					scheduler_proto.Node_STATUS_ONLINE,
					scheduler_proto.Node_STATUS_OFFLINE,
				},
				CpuClass:        scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE,
				CpuAvailable:    3.5,
				MemoryAvailable: 1900,
				GpuClass:        scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
				GpuAvailable:    2,
				DiskClass:       scheduler_proto.DiskClass_DISK_CLASS_SSD,
				DiskAvailable:   5000,
				UsernameLabels:  []string{"username1"},
				NameLabels:      []string{"node1", "node2"},
				CpuLabels:       []string{"Intel i7"},
				GpuLabels:       []string{"Nvidia"},
				MemoryLabels:    []string{"DDR3"},
				DiskLabels:      []string{"Seagate"},
				OtherLabels:     []string{"label1", "label2", "label4"},
				TasksFinished:   100,
				TasksFailed:     80,
				Limit:           10,
			},
		},
		{
			nodes: []*models.Node{
				{
					Name:        "node1",
					Username:    "username1",
					ScheduledAt: &minuteAgo,
				},
				{
					Name:        "node2",
					Username:    "username2",
					ScheduledAt: &hourAgo,
				},
				{
					Name:        "node3",
					Username:    "username3",
					ScheduledAt: &dayAgo,
				},
			},
			nodeNamesExpected:  []string{"node2", "node3"},
			totalCountExpected: 3,
			request: &scheduler_proto.ListNodesRequest{
				Ordering: scheduler_proto.ListNodesRequest_SCHEDULED_AT_DESC,
				Limit:    10,
				Offset:   1,
			},
		},
		{
			nodes: []*models.Node{
				{
					Name:        "node1",
					Username:    "username1",
					ConnectedAt: &minuteAgo,
				},
				{
					Name:        "node2",
					Username:    "username2",
					ConnectedAt: &hourAgo,
				},
				{
					Name:        "node3",
					Username:    "username3",
					ConnectedAt: &dayAgo,
				},
			},
			nodeNamesExpected:  []string{"node1", "node2", "node3"},
			totalCountExpected: 3,
			request:            &scheduler_proto.ListNodesRequest{Limit: 10},
		},
	} {
		for _, node := range testCase.nodes {
			_, err := node.Create(s.db)
			s.Require().NoError(err)
		}
		response, err := s.client.ListNodes(ctx, testCase.request)
		for _, node := range testCase.nodes {
			s.db.Unscoped().Delete(node)
			s.Require().NoError(err)
		}
		s.Require().NoError(err)
		nodeNamesActual := make([]string, len(response.Nodes))
		for i, node := range response.Nodes {
			nodeNamesActual[i] = node.Name
		}
		s.Equal(testCase.nodeNamesExpected, nodeNamesActual)
		s.Equal(testCase.totalCountExpected, response.TotalCount)
	}
}

package models_test

import (
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/assert"

	"github.com/mennanov/scalemate/scheduler/models"
)

func TestSortNodesByStrategy(t *testing.T) {
	for _, testCase := range []struct {
		nodes            []*models.NodeExt
		request          *models.ResourceRequest
		expectedIdsOrder []int64
		strategies       []*scheduler_proto.SchedulingStrategy
	}{
		{
			nodes: []*models.NodeExt{
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 1,
						},
					},
					CpuPrice:    3,
					GpuPrice:    3,
					DiskPrice:   3,
					MemoryPrice: 3,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 2,
						},
					},
					CpuPrice:    2,
					GpuPrice:    2,
					DiskPrice:   2,
					MemoryPrice: 2,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 3,
						},
					},
					CpuPrice:    1,
					GpuPrice:    1,
					DiskPrice:   1,
					MemoryPrice: 1,
				},
			},
			request: &models.ResourceRequest{
				ResourceRequest: scheduler_proto.ResourceRequest{
					Cpu:    1,
					Memory: 1,
					Disk:   1,
					Gpu:    1,
				},
			},
			expectedIdsOrder: []int64{3, 2, 1},
			strategies: []*scheduler_proto.SchedulingStrategy{
				{
					Strategy: scheduler_proto.SchedulingStrategy_CHEAPEST,
				},
			},
		},
		{
			nodes: []*models.NodeExt{
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 1,
						},
					},
					CpuPrice:    120,
					GpuPrice:    120,
					DiskPrice:   120,
					MemoryPrice: 120,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 2,
						},
					},
					CpuPrice:             110,
					GpuPrice:             110,
					DiskPrice:            110,
					MemoryPrice:          110,
					ContainersScheduled:  10,
					ContainersNodeFailed: 2,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 3,
						},
					},
					CpuPrice:             100,
					GpuPrice:             100,
					DiskPrice:            100,
					MemoryPrice:          100,
					ContainersScheduled:  10,
					ContainersNodeFailed: 3,
				},
			},
			request: &models.ResourceRequest{
				ResourceRequest: scheduler_proto.ResourceRequest{
					Cpu:    1,
					Memory: 1,
					Disk:   1,
					Gpu:    1,
				},
			},
			expectedIdsOrder: []int64{2, 3, 1},
			strategies: []*scheduler_proto.SchedulingStrategy{
				{
					Strategy:             scheduler_proto.SchedulingStrategy_CHEAPEST,
					DifferencePercentage: 10,
				},
				{
					Strategy: scheduler_proto.SchedulingStrategy_MOST_RELIABLE,
				},
			},
		},
		{
			nodes: []*models.NodeExt{
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 1,
						},
					},
					CpuPrice:    120,
					GpuPrice:    120,
					DiskPrice:   120,
					MemoryPrice: 120,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id:              2,
							CpuCapacity:     4,
							CpuAvailable:    1,
							GpuCapacity:     2,
							GpuAvailable:    0,
							DiskCapacity:    2,
							DiskAvailable:   1,
							MemoryCapacity:  2,
							MemoryAvailable: 1,
						},
					},
					CpuPrice:             110,
					GpuPrice:             110,
					DiskPrice:            110,
					MemoryPrice:          110,
					ContainersScheduled:  10,
					ContainersNodeFailed: 2,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id:              3,
							CpuCapacity:     4,
							CpuAvailable:    3,
							GpuCapacity:     2,
							GpuAvailable:    0,
							DiskCapacity:    2,
							DiskAvailable:   1,
							MemoryCapacity:  2,
							MemoryAvailable: 1,
						},
					},
					CpuPrice:             105,
					GpuPrice:             105,
					DiskPrice:            105,
					MemoryPrice:          105,
					ContainersScheduled:  10,
					ContainersNodeFailed: 3,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 4,
						},
					},
					CpuPrice:             100,
					GpuPrice:             100,
					DiskPrice:            100,
					MemoryPrice:          100,
					ContainersScheduled:  10,
					ContainersNodeFailed: 4,
				},
			},
			request: &models.ResourceRequest{
				ResourceRequest: scheduler_proto.ResourceRequest{
					Cpu:    1,
					Memory: 1,
					Disk:   1,
					Gpu:    1,
				},
			},
			expectedIdsOrder: []int64{3, 2, 4, 1},
			strategies: []*scheduler_proto.SchedulingStrategy{
				{
					Strategy:             scheduler_proto.SchedulingStrategy_CHEAPEST,
					DifferencePercentage: 10,
				},
				{
					Strategy:             scheduler_proto.SchedulingStrategy_MOST_RELIABLE,
					DifferencePercentage: 50,
				},
				{
					Strategy: scheduler_proto.SchedulingStrategy_LEAST_BUSY,
				},
			},
		},
		{
			nodes: []*models.NodeExt{
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id: 1,
						},
					},
					CpuPrice:    120,
					GpuPrice:    120,
					DiskPrice:   120,
					MemoryPrice: 120,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id:           2,
							CpuCapacity:  8,
							CpuAvailable: 6,
						},
					},
					CpuPrice:             110,
					GpuPrice:             110,
					DiskPrice:            110,
					MemoryPrice:          110,
					ContainersScheduled:  10,
					ContainersNodeFailed: 2,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id:           3,
							CpuCapacity:  8,
							CpuAvailable: 5,
						},
					},
					CpuPrice:             105,
					GpuPrice:             105,
					DiskPrice:            105,
					MemoryPrice:          105,
					ContainersScheduled:  10,
					ContainersNodeFailed: 1,
				},
				{
					Node: &models.Node{
						Node: scheduler_proto.Node{
							Id:           4,
							CpuCapacity:  4,
							CpuAvailable: 0,
						},
					},
					CpuPrice:             100,
					GpuPrice:             100,
					DiskPrice:            100,
					MemoryPrice:          100,
					ContainersScheduled:  10,
					ContainersNodeFailed: 4,
				},
			},
			request: &models.ResourceRequest{
				ResourceRequest: scheduler_proto.ResourceRequest{
					Cpu:    1,
					Memory: 1,
					Disk:   1,
					Gpu:    1,
				},
			},
			expectedIdsOrder: []int64{3, 2, 4, 1},
			strategies: []*scheduler_proto.SchedulingStrategy{
				{
					Strategy:             scheduler_proto.SchedulingStrategy_CHEAPEST,
					DifferencePercentage: 10,
				},
				{
					Strategy:             scheduler_proto.SchedulingStrategy_LEAST_BUSY,
					DifferencePercentage: 50,
				},
				{
					Strategy: scheduler_proto.SchedulingStrategy_MOST_RELIABLE,
				},
			},
		},
		{
			nodes: []*models.NodeExt{},
			request: &models.ResourceRequest{
				ResourceRequest: scheduler_proto.ResourceRequest{
					Cpu:    1,
					Memory: 1,
					Disk:   1,
					Gpu:    1,
				},
			},
			expectedIdsOrder: nil,
			strategies: []*scheduler_proto.SchedulingStrategy{
				{
					Strategy:             scheduler_proto.SchedulingStrategy_CHEAPEST,
					DifferencePercentage: 10,
				},
				{
					Strategy:             scheduler_proto.SchedulingStrategy_LEAST_BUSY,
					DifferencePercentage: 50,
				},
				{
					Strategy: scheduler_proto.SchedulingStrategy_MOST_RELIABLE,
				},
			},
		},
	} {
		models.SortNodesByStrategy(testCase.nodes, testCase.strategies, testCase.request)
		var actualIdsOrder []int64
		for _, node := range testCase.nodes {
			actualIdsOrder = append(actualIdsOrder, node.Id)
		}
		assert.Equal(t, testCase.expectedIdsOrder, actualIdsOrder)
	}
}

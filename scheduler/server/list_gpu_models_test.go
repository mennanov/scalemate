package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestListGpuModels() {
	nodes := []*models.Node{
		{
			Username:     "username1",
			Name:         "node1",
			GpuModel:     "model1",
			GpuClass:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			GpuCapacity:  4,
			GpuAvailable: 2,
			Status:       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username1",
			Name:         "node2",
			GpuModel:     "model1",
			GpuClass:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_ENTRY),
			GpuCapacity:  4,
			GpuAvailable: 1,
			Status:       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username1",
			Name:         "node3",
			GpuModel:     "model2",
			GpuClass:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
			GpuCapacity:  8,
			GpuAvailable: 1,
			Status:       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username2",
			Name:         "node1",
			GpuModel:     "model2",
			GpuClass:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_PRO),
			GpuCapacity:  8,
			GpuAvailable: 1,
			Status:       models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
	}

	for _, node := range nodes {
		_, err := node.Create(s.db)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all GPUs for Nodes online", func(t *testing.T) {
		req := &scheduler_proto.ListGpuModelsRequest{}
		res, err := s.client.ListGpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal([]*scheduler_proto.ListGpuModelsResponse_GpuModel{
			{
				GpuModel:     "model1",
				GpuClass:     scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
				GpuCapacity:  8,
				GpuAvailable: 3,
				NodesCount:   2,
			},
			{
				GpuModel:     "model2",
				GpuClass:     scheduler_proto.GPUClass_GPU_CLASS_PRO,
				GpuCapacity:  8,
				GpuAvailable: 1,
				NodesCount:   1,
			},
		}, res.GpuModels)
	})

	s.T().Run("returns GPUs for requested class", func(t *testing.T) {
		req := &scheduler_proto.ListGpuModelsRequest{
			GpuClass: scheduler_proto.GPUClass_GPU_CLASS_PRO,
		}
		res, err := s.client.ListGpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal([]*scheduler_proto.ListGpuModelsResponse_GpuModel{
			{
				GpuModel:     "model2",
				GpuClass:     scheduler_proto.GPUClass_GPU_CLASS_PRO,
				GpuCapacity:  8,
				GpuAvailable: 1,
				NodesCount:   1,
			},
		}, res.GpuModels)
	})

}

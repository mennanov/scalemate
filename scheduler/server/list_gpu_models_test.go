package server_test

import (
	"context"
	"testing"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
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
		_, err := node.Create(s.service.DB)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all GPUs for Nodes online", func(t *testing.T) {
		req := &scheduler_proto.ListGpuModelsRequest{}
		res, err := s.client.ListGpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(2, len(res.GpuModels))
		mask := fieldmask_utils.MaskFromString("GpuModel,GpuClass,GpuCapacity,GpuAvailable,NodesCount")
		resMap := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.GpuModels[0], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"GpuModel":     "model1",
			"GpuClass":     scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
			"GpuCapacity":  uint32(8),
			"GpuAvailable": uint32(3),
			"NodesCount":   uint32(2),
		}, resMap)

		resMap = make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.GpuModels[1], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"GpuModel":     "model2",
			"GpuClass":     scheduler_proto.GPUClass_GPU_CLASS_PRO,
			"GpuCapacity":  uint32(8),
			"GpuAvailable": uint32(1),
			"NodesCount":   uint32(1),
		}, resMap)
	})

	s.T().Run("returns GPUs for requested class", func(t *testing.T) {
		req := &scheduler_proto.ListGpuModelsRequest{
			GpuClass: scheduler_proto.GPUClass_GPU_CLASS_PRO,
		}
		res, err := s.client.ListGpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(1, len(res.GpuModels))
		mask := fieldmask_utils.MaskFromString("GpuModel,GpuClass,GpuCapacity,GpuAvailable,NodesCount")
		resMap := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.GpuModels[0], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"GpuModel":     "model2",
			"GpuClass":     scheduler_proto.GPUClass_GPU_CLASS_PRO,
			"GpuCapacity":  uint32(8),
			"GpuAvailable": uint32(1),
			"NodesCount":   uint32(1),
		}, resMap)
	})

}

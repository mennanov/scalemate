package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestListCpuModels() {
	nodes := []*models.Node{
		{
			Username:     "username1",
			Name:         "node1",
			CpuModel:     "model1",
			CpuClass:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuCapacity:  4,
			CpuAvailable: 2,
			Status:       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username1",
			Name:         "node2",
			CpuModel:     "model1",
			CpuClass:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuCapacity:  4,
			CpuAvailable: 1,
			Status:       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username1",
			Name:         "node3",
			CpuModel:     "model2",
			CpuClass:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_PRO),
			CpuCapacity:  8,
			CpuAvailable: 1,
			Status:       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username2",
			Name:         "node1",
			CpuModel:     "model2",
			CpuClass:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_PRO),
			CpuCapacity:  8,
			CpuAvailable: 1,
			Status:       models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
	}

	for _, node := range nodes {
		_, err := node.Create(s.service.DB)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all CPUs for Nodes online", func(t *testing.T) {
		req := &scheduler_proto.ListCpuModelsRequest{}
		res, err := s.client.ListCpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(2, len(res.CpuModels))
		mask := fieldmask_utils.MaskFromString("CpuModel,CpuClass,CpuCapacity,CpuAvailable,NodesCount")
		resMap := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.CpuModels[0], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"CpuModel":     "model1",
			"CpuClass":     scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
			"CpuCapacity":  uint32(8),
			"CpuAvailable": float32(3),
			"NodesCount":   uint32(2),
		}, resMap)

		resMap = make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.CpuModels[1], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"CpuModel":     "model2",
			"CpuClass":     scheduler_proto.CPUClass_CPU_CLASS_PRO,
			"CpuCapacity":  uint32(8),
			"CpuAvailable": float32(1),
			"NodesCount":   uint32(1),
		}, resMap)
	})

	s.T().Run("returns CPUs for requested class", func(t *testing.T) {
		req := &scheduler_proto.ListCpuModelsRequest{
			CpuClass: scheduler_proto.CPUClass_CPU_CLASS_PRO,
		}
		res, err := s.client.ListCpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(1, len(res.CpuModels))
		mask := fieldmask_utils.MaskFromString("CpuModel,CpuClass,CpuCapacity,CpuAvailable,NodesCount")
		resMap := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.CpuModels[0], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"CpuModel":     "model2",
			"CpuClass":     scheduler_proto.CPUClass_CPU_CLASS_PRO,
			"CpuCapacity":  uint32(8),
			"CpuAvailable": float32(1),
			"NodesCount":   uint32(1),
		}, resMap)
	})

}

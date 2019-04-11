package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestListCpuModels() {
	nodes := []*models.Node{
		{
			Username:     "username1",
			Name:         "node1",
			CpuModel:     "model1",
			CpuClass:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuCapacity:  4,
			CpuAvailable: 2,
			Status:       utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username1",
			Name:         "node2",
			CpuModel:     "model1",
			CpuClass:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_ENTRY),
			CpuCapacity:  4,
			CpuAvailable: 1,
			Status:       utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username1",
			Name:         "node3",
			CpuModel:     "model2",
			CpuClass:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_PRO),
			CpuCapacity:  8,
			CpuAvailable: 1,
			Status:       utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:     "username2",
			Name:         "node1",
			CpuModel:     "model2",
			CpuClass:     utils.Enum(scheduler_proto.CPUClass_CPU_CLASS_PRO),
			CpuCapacity:  8,
			CpuAvailable: 1,
			Status:       utils.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
	}

	for _, node := range nodes {
		_, err := node.Create(s.db)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all CPUs for Nodes online", func(t *testing.T) {
		req := &scheduler_proto.ListCpuModelsRequest{}
		res, err := s.client.ListCpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal([]*scheduler_proto.ListCpuModelsResponse_CpuModel{
			{
				CpuModel:     "model1",
				CpuClass:     scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
				CpuCapacity:  8,
				CpuAvailable: 3,
				NodesCount:   2,
			},
			{
				CpuModel:     "model2",
				CpuClass:     scheduler_proto.CPUClass_CPU_CLASS_PRO,
				CpuCapacity:  8,
				CpuAvailable: 1,
				NodesCount:   1,
			},
		}, res.CpuModels)
	})

	s.T().Run("returns CPUs for requested class", func(t *testing.T) {
		req := &scheduler_proto.ListCpuModelsRequest{
			CpuClass: scheduler_proto.CPUClass_CPU_CLASS_PRO,
		}
		res, err := s.client.ListCpuModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(1, len(res.CpuModels))
		s.Equal([]*scheduler_proto.ListCpuModelsResponse_CpuModel{
			{
				CpuModel:     "model2",
				CpuClass:     scheduler_proto.CPUClass_CPU_CLASS_PRO,
				CpuCapacity:  8,
				CpuAvailable: 1,
				NodesCount:   1,
			},
		}, res.CpuModels)
	})

}

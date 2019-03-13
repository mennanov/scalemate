package server_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestListMemoryModels() {
	nodes := []*models.Node{
		{
			Username:        "username1",
			Name:            "node1",
			MemoryModel:     "model1",
			MemoryCapacity:  4,
			MemoryAvailable: 2,
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:        "username1",
			Name:            "node2",
			MemoryModel:     "model1",
			MemoryCapacity:  4,
			MemoryAvailable: 1,
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:        "username1",
			Name:            "node3",
			MemoryModel:     "model2",
			MemoryCapacity:  8,
			MemoryAvailable: 1,
			Status:          models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:        "username2",
			Name:            "node1",
			MemoryModel:     "model2",
			MemoryCapacity:  8,
			MemoryAvailable: 1,
			Status:          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
	}

	for _, node := range nodes {
		_, err := node.Create(s.db)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all GPUs for Nodes online", func(t *testing.T) {
		req := &empty.Empty{}
		res, err := s.client.ListMemoryModels(ctx, req)
		s.Require().NoError(err)
		s.Equal([]*scheduler_proto.ListMemoryModelsResponse_MemoryModel{
			{
				MemoryModel:     "model1",
				MemoryCapacity:  8,
				MemoryAvailable: 3,
				NodesCount:      2,
			},
			{
				MemoryModel:     "model2",
				MemoryCapacity:  8,
				MemoryAvailable: 1,
				NodesCount:      1,
			},
		}, res.MemoryModels)
	})
}

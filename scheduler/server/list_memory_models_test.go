package server_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
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
		_, err := node.Create(s.service.DB)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all GPUs for Nodes online", func(t *testing.T) {
		req := &empty.Empty{}
		res, err := s.client.ListMemoryModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(2, len(res.MemoryModels))
		mask := fieldmask_utils.MaskFromString("MemoryModel,MemoryCapacity,MemoryAvailable,NodesCount")
		resMap := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.MemoryModels[0], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"MemoryModel":     "model1",
			"MemoryCapacity":  uint32(8),
			"MemoryAvailable": uint32(3),
			"NodesCount":      uint32(2),
		}, resMap)

		resMap = make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.MemoryModels[1], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"MemoryModel":     "model2",
			"MemoryCapacity":  uint32(8),
			"MemoryAvailable": uint32(1),
			"NodesCount":      uint32(1),
		}, resMap)
	})
}

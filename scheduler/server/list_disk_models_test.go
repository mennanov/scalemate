package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestListDiskModels() {
	nodes := []*models.Node{
		{
			Username:      "username1",
			Name:          "node1",
			DiskModel:     "model1",
			DiskClass:     utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskCapacity:  4,
			DiskAvailable: 2,
			Status:        utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:      "username1",
			Name:          "node2",
			DiskModel:     "model1",
			DiskClass:     utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskCapacity:  4,
			DiskAvailable: 1,
			Status:        utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:      "username1",
			Name:          "node3",
			DiskModel:     "model2",
			DiskClass:     utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
			DiskCapacity:  8,
			DiskAvailable: 1,
			Status:        utils.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:      "username2",
			Name:          "node1",
			DiskModel:     "model2",
			DiskClass:     utils.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
			DiskCapacity:  8,
			DiskAvailable: 1,
			Status:        utils.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
	}

	for _, node := range nodes {
		_, err := node.Create(s.db)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all Disks for Nodes online", func(t *testing.T) {
		req := &scheduler_proto.ListDiskModelsRequest{}
		res, err := s.client.ListDiskModels(ctx, req)
		s.Require().NoError(err)
		s.Equal([]*scheduler_proto.ListDiskModelsResponse_DiskModel{
			{
				DiskModel:     "model1",
				DiskClass:     scheduler_proto.DiskClass_DISK_CLASS_HDD,
				DiskCapacity:  8,
				DiskAvailable: 3,
				NodesCount:    2,
			},
			{
				DiskModel:     "model2",
				DiskClass:     scheduler_proto.DiskClass_DISK_CLASS_SSD,
				DiskCapacity:  8,
				DiskAvailable: 1,
				NodesCount:    1,
			},
		}, res.DiskModels)
	})

	s.T().Run("returns Disks for requested class", func(t *testing.T) {
		req := &scheduler_proto.ListDiskModelsRequest{
			DiskClass: scheduler_proto.DiskClass_DISK_CLASS_SSD,
		}
		res, err := s.client.ListDiskModels(ctx, req)
		s.Require().NoError(err)
		s.Equal([]*scheduler_proto.ListDiskModelsResponse_DiskModel{
			{
				DiskModel:     "model2",
				DiskClass:     scheduler_proto.DiskClass_DISK_CLASS_SSD,
				DiskCapacity:  8,
				DiskAvailable: 1,
				NodesCount:    1,
			},
		}, res.DiskModels)
	})

}

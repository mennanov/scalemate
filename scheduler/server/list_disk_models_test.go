package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestListDiskModels() {
	nodes := []*models.Node{
		{
			Username:      "username1",
			Name:          "node1",
			DiskModel:     "model1",
			DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskCapacity:  4,
			DiskAvailable: 2,
			Status:        models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:      "username1",
			Name:          "node2",
			DiskModel:     "model1",
			DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
			DiskCapacity:  4,
			DiskAvailable: 1,
			Status:        models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:      "username1",
			Name:          "node3",
			DiskModel:     "model2",
			DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
			DiskCapacity:  8,
			DiskAvailable: 1,
			Status:        models.Enum(scheduler_proto.Node_STATUS_ONLINE),
		},
		{
			Username:      "username2",
			Name:          "node1",
			DiskModel:     "model2",
			DiskClass:     models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
			DiskCapacity:  8,
			DiskAvailable: 1,
			Status:        models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		},
	}

	for _, node := range nodes {
		_, err := node.Create(s.service.DB)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns all Disks for Nodes online", func(t *testing.T) {
		req := &scheduler_proto.ListDiskModelsRequest{}
		res, err := s.client.ListDiskModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(2, len(res.DiskModels))
		mask := fieldmask_utils.MaskFromString("DiskModel,DiskClass,DiskCapacity,DiskAvailable,NodesCount")
		resMap := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.DiskModels[0], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"DiskModel":     "model1",
			"DiskClass":     scheduler_proto.DiskClass_DISK_CLASS_HDD,
			"DiskCapacity":  uint32(8),
			"DiskAvailable": uint32(3),
			"NodesCount":    uint32(2),
		}, resMap)

		resMap = make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.DiskModels[1], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"DiskModel":     "model2",
			"DiskClass":     scheduler_proto.DiskClass_DISK_CLASS_SSD,
			"DiskCapacity":  uint32(8),
			"DiskAvailable": uint32(1),
			"NodesCount":    uint32(1),
		}, resMap)
	})

	s.T().Run("returns Disks for requested class", func(t *testing.T) {
		req := &scheduler_proto.ListDiskModelsRequest{
			DiskClass: scheduler_proto.DiskClass_DISK_CLASS_SSD,
		}
		res, err := s.client.ListDiskModels(ctx, req)
		s.Require().NoError(err)
		s.Equal(1, len(res.DiskModels))
		mask := fieldmask_utils.MaskFromString("DiskModel,DiskClass,DiskCapacity,DiskAvailable,NodesCount")
		resMap := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, res.DiskModels[0], resMap, stringEye, stringEye)
		s.Require().NoError(err)
		s.Equal(map[string]interface{}{
			"DiskModel":     "model2",
			"DiskClass":     scheduler_proto.DiskClass_DISK_CLASS_SSD,
			"DiskCapacity":  uint32(8),
			"DiskAvailable": uint32(1),
			"NodesCount":    uint32(1),
		}, resMap)
	})

}

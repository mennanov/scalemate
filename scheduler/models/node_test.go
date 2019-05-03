package models_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ModelsTestSuite) TestNode_Create() {
	for _, node := range []*models.Node{
		{
			Node: scheduler_proto.Node{
				Id:                     1,
				Username:               "username",
				Name:                   "name",
				Status:                 0,
				CpuCapacity:            0,
				CpuAvailable:           0,
				CpuClass:               0,
				MemoryCapacity:         0,
				MemoryAvailable:        0,
				GpuCapacity:            0,
				GpuAvailable:           0,
				GpuClass:               0,
				DiskCapacity:           0,
				DiskAvailable:          0,
				DiskClass:              0,
				NetworkIngressCapacity: 0,
				NetworkEgressCapacity:  0,
				Labels:                 nil,
				ContainersFinished:     0,
				ContainersFailed:       0,
				ConnectedAt:            nil,
				DisconnectedAt:         nil,
				LastScheduledAt:        nil,
				CreatedAt:              time.Time{},
				UpdatedAt:              nil,
				Ip:                     nil,
				Fingerprint:            nil,
				XXX_NoUnkeyedLiteral:   struct{}{},
				XXX_unrecognized:       nil,
				XXX_sizecache:          0,
			},
		},
		new(models.Node),
	} {
		_, err := node.Create(s.db)
		s.Require().NoError(err)
		s.NotNil(node.Id)
		s.False(node.CreatedAt.IsZero())
		s.Nil(node.UpdatedAt)
		nodeFromDB, err := models.NewNodeFromDB(s.db, node.Id, false, s.logger)
		s.Require().NoError(err)
		s.Equal(node, nodeFromDB)
	}
}

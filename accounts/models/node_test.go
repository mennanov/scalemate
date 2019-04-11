package models_test

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ModelsTestSuite) TestNode_FromSchedulerProto() {
	for _, schedulerNodeProto := range []*scheduler_proto.Node{
		{
			Id:          1,
			Username:    "username",
			Name:        "node_name",
			CpuModel:    "cpu model",
			MemoryModel: "memory model",
			GpuModel:    "gpu model",
			DiskModel:   "disk model",
		},
		{
			Id:          2,
			Username:    "username",
			Name:        "node_name",
			CpuModel:    "cpu model",
			MemoryModel: "memory model",
			GpuModel:    "gpu model",
			DiskModel:   "disk model",
			CreatedAt: &timestamp.Timestamp{
				Seconds: 1,
				Nanos:   0,
			},
			UpdatedAt: &timestamp.Timestamp{
				Seconds: 2,
				Nanos:   0,
			},
		},
	} {
		node := &models.Node{}
		node.FromSchedulerProto(schedulerNodeProto)
		s.Equal(schedulerNodeProto.Id, node.ID)
		s.Equal(schedulerNodeProto.Username, node.Username)
		s.Equal(schedulerNodeProto.CpuModel, node.CpuModel)
		s.Equal(schedulerNodeProto.MemoryModel, node.MemoryModel)
		s.Equal(schedulerNodeProto.GpuModel, node.GpuModel)
		s.Equal(schedulerNodeProto.DiskModel, node.DiskModel)
		// Node's CreatedAt and UpdatedAt fields should not be populated from proto.
		s.True(node.CreatedAt.IsZero())
		s.Nil(node.UpdatedAt)
	}
}

func (s *ModelsTestSuite) TestNode_ToProto() {
	updatedAt := time.Unix(2, 0)
	for _, node := range []*models.Node{
		{
			Model: utils.Model{
				ID:        1,
				CreatedAt: time.Unix(1, 0),
				UpdatedAt: nil,
				DeletedAt: nil,
			},
			Username:    "username",
			Name:        "name",
			CpuModel:    "cpu model",
			MemoryModel: "memory model",
			GpuModel:    "gpu model",
			DiskModel:   "disk model",
		},
		{
			Model: utils.Model{
				ID:        1,
				CreatedAt: time.Unix(0, 0),
				UpdatedAt: &updatedAt,
				DeletedAt: nil,
			},
			Username:    "username",
			Name:        "name",
			CpuModel:    "cpu model",
			MemoryModel: "memory model",
			GpuModel:    "gpu model",
			DiskModel:   "disk model",
		},
	} {
		nodeProto, err := node.ToProto(nil)
		s.Require().NoError(err)
		s.Equal(node.ID, nodeProto.Id)
		s.Equal(node.Username, nodeProto.Username)
		s.Equal(node.Name, nodeProto.Name)
		s.Equal(node.CpuModel, nodeProto.CpuModel)
		s.Equal(node.MemoryModel, nodeProto.MemoryModel)
		s.Equal(node.GpuModel, nodeProto.GpuModel)
		s.Equal(node.DiskModel, nodeProto.DiskModel)
		if node.CreatedAt.IsZero() {
			s.Nil(nodeProto.CreatedAt)
		} else {
			createdAt, err := ptypes.TimestampProto(node.CreatedAt)
			s.Require().NoError(err)
			s.Equal(createdAt, nodeProto.CreatedAt)
		}
		if node.UpdatedAt == nil {
			s.Nil(nodeProto.UpdatedAt)
		} else {
			updatedAt, err := ptypes.TimestampProto(*node.UpdatedAt)
			s.Require().NoError(err)
			s.Equal(updatedAt, nodeProto.UpdatedAt)
		}
	}
}

func (s *ModelsTestSuite) TestNode_Create() {
	node := &models.Node{
		Model: utils.Model{
			ID: 1,
		},
		Username:    "username",
		Name:        "name",
		CpuModel:    "cpu model",
		MemoryModel: "memory model",
		GpuModel:    "gpu model",
		DiskModel:   "disk model",
	}
	event, err := node.Create(s.db)
	s.Require().NoError(err)
	s.NotNil(event)
	nodeFromDb := &models.Node{}
	s.Require().NoError(nodeFromDb.Get(s.db, node.Username, node.Name))
	s.Equal(node, nodeFromDb)
}

package models_test

import (
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ModelsTestSuite) TestNode_FromSchedulerProto() {
	updated := time.Now().Add(time.Second)
	for _, schedulerNodeProto := range []*scheduler_proto.Node{
		{
			Id:       1,
			Username: "username",
			Name:     "node_name",
		},
		{
			Id:        2,
			Username:  "username",
			Name:      "node_name",
			CreatedAt: time.Now(),
			UpdatedAt: &updated,
		},
	} {
		node := models.NewNodeFromSchedulerProto(schedulerNodeProto)
		s.Equal(schedulerNodeProto.Id, node.ID)
		s.Equal(schedulerNodeProto.Fingerprint, node.Fingerprint)
		s.Equal(schedulerNodeProto.Name, node.Name)
		s.Equal(schedulerNodeProto.Username, node.Username)
		// Node's CreatedAt and UpdatedAt fields should not be populated from proto.
		s.True(node.CreatedAt.IsZero())
		s.Nil(node.UpdatedAt)
	}
}

func (s *ModelsTestSuite) TestNode_Create() {
	node := &models.Node{
		Model: utils.Model{
			ID: 1,
		},
		Username: "username",
		Name:     "name",
	}
	s.Require().NoError(node.Create(s.db))

	// Verify that the struct fields are updated from DB.
	s.False(node.CreatedAt.IsZero())
	// UpdatedAt is still nil as the node has never been updated.
	s.Nil(node.UpdatedAt)

	nodeFromDb, err := models.NodeLookUp(s.db, node.Username, node.Name)
	s.Require().NoError(err)
	s.Equal(node, nodeFromDb)
}

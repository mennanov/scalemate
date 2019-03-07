package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ServerTestSuite) TestGetNode() {
	node := &models.Node{
		Username: "test_username",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)
	s.Require().NotNil(node.ID)

	ctx := context.Background()

	s.T().Run("successful lookup", func(t *testing.T) {
		response, err := s.client.GetNode(ctx, &scheduler_proto.NodeLookupRequest{
			NodeId: node.ID,
		})
		s.Require().NoError(err)
		s.Equal(node.ID, response.Id)
	})

	s.T().Run("not found", func(t *testing.T) {
		response, err := s.client.GetNode(ctx, &scheduler_proto.NodeLookupRequest{
			NodeId: node.ID + 1, // Non-existing Node ID.
		})
		s.Nil(response)
		s.assertGRPCError(err, codes.NotFound)
	})

}

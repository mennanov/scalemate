package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
)

func (s *ServerTestSuite) TestGetTask() {
	job := &models.Job{
		Username: "username",
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)
	s.Require().NotNil(job.ID)

	node := &models.Node{
		Username: "node_username",
	}
	_, err = node.Create(s.db)
	s.Require().NoError(err)
	s.Require().NotNil(node.ID)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
	}
	_, err = task.Create(s.db)
	s.Require().NoError(err)
	s.Require().NotNil(task.ID)

	ctx := context.Background()
	req := &scheduler_proto.GetTaskRequest{
		TaskId: task.ID,
	}

	s.T().Run("successful for Task owner", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: job.Username})
		defer restoreClaims()

		response, err := s.client.GetTask(ctx, req)
		s.Require().NoError(err)
		s.Equal(task.ID, response.Id)
	})

	s.T().Run("successful for admin", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(
			&auth.Claims{Username: "admin", Role: accounts_proto.User_ADMIN})
		defer restoreClaims()

		response, err := s.client.GetTask(ctx, req)
		s.Require().NoError(err)
		s.Equal(task.ID, response.Id)
	})

	s.T().Run("fails for other non-admin", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: "unknown"})
		defer restoreClaims()

		response, err := s.client.GetTask(ctx, req)
		s.assertGRPCError(err, codes.PermissionDenied)
		s.Nil(response)
	})

}

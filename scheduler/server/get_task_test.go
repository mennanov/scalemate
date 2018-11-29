package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
)

func (s *ServerTestSuite) TestGetTask() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	job := &models.Job{
		Username: "username",
	}
	_, err := job.Create(s.service.DB)
	s.Require().NoError(err)
	s.Require().NotNil(job.ID)

	node := &models.Node{
		Username: "node_username",
	}
	_, err = node.Create(s.service.DB)
	s.Require().NoError(err)
	s.Require().NotNil(node.ID)

	task := &models.Task{
		JobID:  job.ID,
		NodeID: node.ID,
	}
	_, err = task.Create(s.service.DB)
	s.Require().NoError(err)
	s.Require().NotNil(task.ID)

	ctx := context.Background()
	req := &scheduler_proto.GetTaskRequest{
		TaskId: task.ID,
	}
	accountsClient := NewMockAccountsClient(ctrl)

	s.T().Run("successful for Task owner", func(t *testing.T) {
		accessToken := s.createToken("username", "", accounts_proto.User_USER, "access", time.Minute)
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		response, err := s.client.GetTask(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(task.ID, response.Id)
	})

	s.T().Run("successful for admin", func(t *testing.T) {
		accessToken := s.createToken("username2", "", accounts_proto.User_ADMIN, "access", time.Minute)
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		response, err := s.client.GetTask(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(task.ID, response.Id)
	})

	s.T().Run("fails for other non-admin", func(t *testing.T) {
		accessToken := s.createToken("username3", "", accounts_proto.User_USER, "access", time.Minute)
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		response, err := s.client.GetTask(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.assertGRPCError(err, codes.PermissionDenied)
		s.Nil(response)
	})

}

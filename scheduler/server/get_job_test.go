package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *ServerTestSuite) TestGetJob() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	job := &models.Job{
		Username: "username",
	}
	_, err := job.Create(s.service.DB)
	s.Require().NoError(err)
	s.Require().NotNil(job.ID)

	ctx := context.Background()
	req := &scheduler_proto.GetJobRequest{
		JobId: job.ID,
	}
	accountsClient := NewMockAccountsClient(ctrl)

	s.T().Run("successful for Job owner", func(t *testing.T) {
		accessToken := s.createToken("username", "", accounts_proto.User_USER, "access", time.Minute)
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.GetJob(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(job.ID, res.Id)
	})

	s.T().Run("successful for admin", func(t *testing.T) {
		accessToken := s.createToken("username2", "", accounts_proto.User_ADMIN, "access", time.Minute)
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.GetJob(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(job.ID, res.Id)
	})

	s.T().Run("fails for other non-admin", func(t *testing.T) {
		accessToken := s.createToken("username3", "", accounts_proto.User_USER, "access", time.Minute)
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.GetJob(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.assertGRPCError(err, codes.PermissionDenied)
		s.Nil(res)
	})

}

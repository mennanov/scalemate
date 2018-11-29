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

func (s *ServerTestSuite) TestListJobs() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	job1 := &models.Job{
		Username: "username1",
		Status:   models.Enum(scheduler_proto.Job_STATUS_PENDING),
	}
	_, err := job1.Create(s.service.DB)
	s.Require().NoError(err)
	s.Require().NotNil(job1.ID)

	job2 := &models.Job{
		Username: "username1",
		Status:   models.Enum(scheduler_proto.Job_STATUS_FINISHED),
	}
	_, err = job2.Create(s.service.DB)
	s.Require().NoError(err)
	s.Require().NotNil(job2.ID)

	job3 := &models.Job{
		Username: "username2",
	}
	_, err = job3.Create(s.service.DB)
	s.Require().NoError(err)
	s.Require().NotNil(job3.ID)

	ctx := context.Background()
	accountsClient := NewMockAccountsClient(ctrl)

	s.T().Run("returns owned Jobs", func(t *testing.T) {
		accessToken := s.createToken("username1", "", accounts_proto.User_USER, "access", time.Minute)
		req := &scheduler_proto.ListJobsRequest{
			Username: "username1",
		}
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.ListJobs(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(uint32(2), res.TotalCount)
		s.Equal(job2.ID, res.Jobs[0].Id)
		s.Equal(job1.ID, res.Jobs[1].Id)
	})

	s.T().Run("returns all Jobs for admin", func(t *testing.T) {
		accessToken := s.createToken("username2", "", accounts_proto.User_ADMIN, "access", time.Minute)
		req := &scheduler_proto.ListJobsRequest{
			Username: "username1",
		}
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.ListJobs(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(uint32(2), res.TotalCount)
		s.Equal(job2.ID, res.Jobs[0].Id)
		s.Equal(job1.ID, res.Jobs[1].Id)
	})

	s.T().Run("permission denied for other username", func(t *testing.T) {
		accessToken := s.createToken("username2", "", accounts_proto.User_USER, "access", time.Minute)
		req := &scheduler_proto.ListJobsRequest{
			Username: "username1",
		}
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.ListJobs(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.assertGRPCError(err, codes.PermissionDenied)
		s.Nil(res)
	})

	s.T().Run("returns Jobs for requested status", func(t *testing.T) {
		accessToken := s.createToken("username1", "", accounts_proto.User_USER, "access", time.Minute)
		req := &scheduler_proto.ListJobsRequest{
			Username: "username1",
			Status:   []scheduler_proto.Job_Status{scheduler_proto.Job_STATUS_PENDING},
		}
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.ListJobs(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(uint32(1), res.TotalCount)
		s.Equal(job1.ID, res.Jobs[0].Id)
	})

	s.T().Run("returns Jobs for requested statuses and order", func(t *testing.T) {
		accessToken := s.createToken("username1", "", accounts_proto.User_USER, "access", time.Minute)
		req := &scheduler_proto.ListJobsRequest{
			Username: "username1",
			Status: []scheduler_proto.Job_Status{
				scheduler_proto.Job_STATUS_PENDING,
				scheduler_proto.Job_STATUS_FINISHED,
			},
			Ordering: scheduler_proto.ListJobsRequest_CREATED_AT_ASC,
		}
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.ListJobs(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(uint32(2), res.TotalCount)
		s.Equal(job1.ID, res.Jobs[0].Id)
		s.Equal(job2.ID, res.Jobs[1].Id)
	})

	s.T().Run("returns Jobs with limit and offset", func(t *testing.T) {
		accessToken := s.createToken("username1", "", accounts_proto.User_USER, "access", time.Minute)
		req := &scheduler_proto.ListJobsRequest{
			Username: "username1",
			Limit:    1,
			Offset:   1,
		}
		jwtCredentials := auth.NewJWTCredentials(
			accountsClient, &accounts_proto.AuthTokens{AccessToken: accessToken}, tokensFakeSaver)

		res, err := s.client.ListJobs(ctx, req, grpc.PerRPCCredentials(jwtCredentials))
		s.Require().NoError(err)
		s.Equal(uint32(2), res.TotalCount)
		s.Equal(1, len(res.Jobs))
		s.Equal(job1.ID, res.Jobs[0].Id)
	})

}

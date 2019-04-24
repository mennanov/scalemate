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

func (s *ServerTestSuite) TestGetJob() {
	job := &models.Container{
		Username: "test_username",
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)
	s.Require().NotNil(job.ID)

	ctx := context.Background()
	req := &scheduler_proto.JobLookupRequest{
		JobId: job.ID,
	}

	s.T().Run("successful for Container owner", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: job.Username})
		defer restoreClaims()

		res, err := s.client.GetJob(ctx, req)
		s.Require().NoError(err)
		s.Equal(job.ID, res.Id)
	})

	s.T().Run("successful for admin", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(
			&auth.Claims{Username: "admin", Role: accounts_proto.User_ADMIN})
		defer restoreClaims()

		res, err := s.client.GetJob(ctx, req)
		s.Require().NoError(err)
		s.Equal(job.ID, res.Id)
	})

	s.T().Run("fails for other non-admin", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: "unknown"})
		defer restoreClaims()

		res, err := s.client.GetJob(ctx, req)
		s.assertGRPCError(err, codes.PermissionDenied)
		s.Nil(res)
	})

}

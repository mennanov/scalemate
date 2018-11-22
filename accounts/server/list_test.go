package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/accounts/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func (s *ServerTestSuite) TestList() {
	user1 := s.createTestUser(&models.User{
		Username: "aaa",
		Email:    "aaa@mail.com",
		Role:     accounts_proto.User_USER,
		Banned:   false,
	}, "password")
	user2 := s.createTestUser(&models.User{
		Username: "bbb",
		Email:    "bbb@mail.com",
		Role:     accounts_proto.User_ADMIN,
		Banned:   false,
	}, "password")
	user3 := s.createTestUser(&models.User{
		Username: "ccc",
		Email:    "ccc@mail.com",
		Role:     accounts_proto.User_ADMIN,
		Banned:   true,
	}, "password")

	s.T().Run("ListAll", func(t *testing.T) {
		ctx := context.Background()
		req := &accounts_proto.ListUsersRequest{}

		res, err := s.client.List(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
		s.Require().NoError(err)
		require.NotNil(t, res)

		assert.Equal(t, 3, len(res.User))
		assert.Equal(t, uint32(3), res.TotalCount)
		assert.Equal(t, user3.Username, res.User[0].Username)
		assert.Equal(t, user2.Username, res.User[1].Username)
		assert.Equal(t, user1.Username, res.User[2].Username)
	})

	s.T().Run("ListBannedOnly", func(t *testing.T) {
		ctx := context.Background()
		req := &accounts_proto.ListUsersRequest{Banned: &accounts_proto.ListUsersRequest_BannedOnly{BannedOnly: true}}

		res, err := s.client.List(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
		s.Require().NoError(err)
		require.NotNil(t, res)

		assert.Equal(t, 1, len(res.User))
		assert.Equal(t, uint32(1), res.TotalCount)
		assert.Equal(t, user3.Username, res.User[0].Username)
	})

	s.T().Run("ListActiveOnly Order by CreatedAt Limit 1 Offset 1", func(t *testing.T) {
		ctx := context.Background()
		req := &accounts_proto.ListUsersRequest{
			Banned: &accounts_proto.ListUsersRequest_ActiveOnly{ActiveOnly: true},
			Ordering: []accounts_proto.ListUsersRequest_Ordering{
				accounts_proto.ListUsersRequest_CREATED_AT_ASC,
				accounts_proto.ListUsersRequest_EMAIL_ASC,
			},
			Role:   []accounts_proto.User_Role{accounts_proto.User_USER, accounts_proto.User_ADMIN},
			Limit:  1,
			Offset: 1,
		}

		res, err := s.client.List(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
		s.Require().NoError(err)
		require.NotNil(t, res)

		require.Equal(t, 1, len(res.User))
		assert.Equal(t, uint32(2), res.TotalCount)
		assert.Equal(t, user2.Username, res.User[0].Username)
	})

	s.T().Run("List With Offset Out Of Range", func(t *testing.T) {
		ctx := context.Background()
		req := &accounts_proto.ListUsersRequest{
			Limit:  10,
			Offset: 10,
		}

		res, err := s.client.List(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
		s.Require().NoError(err)
		require.NotNil(t, res)

		assert.Equal(t, uint32(3), res.TotalCount)
	})
}

func (s *ServerTestSuite) TestListUnauthenticated() {
	ctx := context.Background()
	req := &accounts_proto.ListUsersRequest{}

	// Make a request with no access credentials.
	_, err := s.client.List(ctx, req)
	s.assertGRPCError(err, codes.Unauthenticated)
}

func (s *ServerTestSuite) TestListPermissionDenied() {
	ctx := context.Background()
	req := &accounts_proto.ListUsersRequest{}

	// Make a request with insufficient access credentials.
	_, err := s.client.List(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_USER))
	s.assertGRPCError(err, codes.PermissionDenied)
}

package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/accounts/models"
	"google.golang.org/grpc/codes"
)

func (s *ServerTestSuite) TestPasswordAuth() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res, user, time.Now(), "")
}

func (s *ServerTestSuite) TestPasswordAuthIncorrectPassword() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "incorrect password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.assertGRPCError(err, codes.InvalidArgument)
	s.Nil(res)
}

func (s *ServerTestSuite) TestPasswordAuthBannedUser() {
	user := s.createTestUser(&models.User{
		Username: "username",
		Banned:   true,
	}, "password")

	ctx := context.Background()
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.assertGRPCError(err, codes.PermissionDenied)
	s.Nil(res)
}

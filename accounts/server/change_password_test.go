package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestChangePassword() {
	messages, err := events.NewAMQPRawConsumer(s.amqpChannel, events.AccountsAMQPExchangeName, "", "#")
	s.Require().NoError(err)

	user := s.createTestUserQuick("password")
	originalPasswordHash := user.PasswordHash

	ctx := context.Background()
	req := &accounts_proto.ChangePasswordRequest{
		Username: user.Username,
		Password: "new password",
	}

	res, err := s.client.ChangePassword(ctx, req, s.userAccessCredentials(user, time.Minute))
	s.Require().NoError(err)
	s.NotNil(res)

	// Verify that the user's password is updated.
	err = user.LookUp(s.db, &accounts_proto.UserLookupRequest{Id: uint32(user.ID)})
	s.Require().NoError(err)
	s.NotEqual(user.PasswordHash, originalPasswordHash)
	utils.WaitForMessages(messages, "accounts.user.updated.*?password_changed_at.*?")
}

func (s *ServerTestSuite) TestChangePasswordLookupFails() {
	ctx := context.Background()
	req := &accounts_proto.ChangePasswordRequest{
		Username: "nonexisting_username",
		Password: "new password",
	}

	_, err := s.client.ChangePassword(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.assertGRPCError(err, codes.NotFound)
}

func (s *ServerTestSuite) TestChangePasswordInvalidPassword() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()
	req := &accounts_proto.ChangePasswordRequest{
		Username: user.Username,
		Password: " ",
	}

	_, err := s.client.ChangePassword(ctx, req, s.userAccessCredentials(user, time.Minute))
	s.assertGRPCError(err, codes.InvalidArgument)
}

func (s *ServerTestSuite) TestChangePasswordUnauthenticated() {
	ctx := context.Background()
	req := &accounts_proto.ChangePasswordRequest{}

	// Make a request with no access credentials.
	_, err := s.client.ChangePassword(ctx, req)
	s.assertGRPCError(err, codes.Unauthenticated)
}

func (s *ServerTestSuite) TestChangePasswordPermissionDenied() {
	ctx := context.Background()
	req := &accounts_proto.ChangePasswordRequest{
		Username: "username",
		Password: "password",
	}

	// Make a request with insufficient access credentials.
	_, err := s.client.ChangePassword(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_USER))
	s.assertGRPCError(err, codes.PermissionDenied)
}

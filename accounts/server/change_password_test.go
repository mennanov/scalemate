package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events"
)

func (s *ServerTestSuite) TestChangePassword() {
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
	err = user.LookUp(s.db, &accounts_proto.UserLookupRequest{Id: user.ID})
	s.Require().NoError(err)
	s.NotEqual(user.PasswordHash, originalPasswordHash)
	s.NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_UPDATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{Id: user.ID},
		},
	})))
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

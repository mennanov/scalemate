package server_test

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *ServerTestSuite) TestChangePassword() {
	ctx := context.Background()
	// Register a new user.
	registerRequest := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "email@mail.com",
		Password: "password",
	}
	registeredUser, err := s.client.Register(ctx, registerRequest)
	s.Require().NoError(err)

	// Authenticate the user.
	authTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
		Request: &accounts_proto.PasswordAuthRequest_UserAuth{
			UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
				Username: registerRequest.Username,
				Password: registerRequest.Password,
			},
		},
	})
	s.Require().NoError(err)

	creds := grpc.PerRPCCredentials(auth.NewSimpleJWTCredentials(authTokens.AccessToken))

	s.Run("change password then authenticate succeeds", func() {
		s.T().Parallel()
		newPassword := "new password"
		_, err = s.client.ChangePassword(ctx, &accounts_proto.ChangePasswordRequest{
			Username: registeredUser.Username,
			Password: newPassword,
		}, creds)
		s.Require().NoError(err)

		s.NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
			Type:    events_proto.Event_UPDATED,
			Service: events_proto.Service_ACCOUNTS,
			Payload: &events_proto.Event_AccountsUser{
				AccountsUser: &accounts_proto.User{Id: registeredUser.Id},
			},
		})))

		// Authenticate the user with the new password.
		newAuthTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
			Request: &accounts_proto.PasswordAuthRequest_UserAuth{
				UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
					Username: registerRequest.Username,
					Password: newPassword,
				},
			},
		})
		s.Require().NoError(err)
		s.NotNil(newAuthTokens)
	})

	s.Run("not found for non-existing user", func() {
		s.T().Parallel()
		_, err = s.client.ChangePassword(ctx, &accounts_proto.ChangePasswordRequest{
			Username: "non_existing_username",
			Password: "new password",
		}, creds)
	})
}

//
//func (s *ServerTestSuite) TestChangePasswordLookupFails() {
//	ctx := context.Background()
//	req := &accounts_proto.ChangePasswordRequest{
//		Username: "nonexisting_username",
//		Password: "new password",
//	}
//
//	_, err := s.client.ChangePassword(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
//	s.assertGRPCError(err, codes.NotFound)
//}
//
//func (s *ServerTestSuite) TestChangePasswordInvalidPassword() {
//	user := s.createTestUserQuick("password")
//
//	ctx := context.Background()
//	req := &accounts_proto.ChangePasswordRequest{
//		Username: user.Username,
//		Password: " ",
//	}
//
//	_, err := s.client.ChangePassword(ctx, req, s.userAccessCredentials(user, time.Minute))
//	s.assertGRPCError(err, codes.InvalidArgument)
//}
//
//func (s *ServerTestSuite) TestChangePasswordUnauthenticated() {
//	ctx := context.Background()
//	req := &accounts_proto.ChangePasswordRequest{}
//
//	// Make a request with no access credentials.
//	_, err := s.client.ChangePassword(ctx, req)
//	s.assertGRPCError(err, codes.Unauthenticated)
//}
//
//func (s *ServerTestSuite) TestChangePasswordPermissionDenied() {
//	ctx := context.Background()
//	req := &accounts_proto.ChangePasswordRequest{
//		Username: "username",
//		Password: "password",
//	}
//
//	// Make a request with insufficient access credentials.
//	_, err := s.client.ChangePassword(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_USER))
//	s.assertGRPCError(err, codes.PermissionDenied)
//}

package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
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
		wait := testutils.ExpectMessages(s.sc, events.AccountsSubjectName, s.logger, "Event_AccountsUserPasswordChanged")
		_, err = s.client.ChangePassword(ctx, &accounts_proto.ChangePasswordRequest{
			Username: registeredUser.Username,
			Password: newPassword,
		}, creds)
		s.Require().NoError(err)
		s.Require().NoError(wait(time.Second))

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
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
	})

	s.Run("invalid arguments for invalid requests", func() {
		s.T().Parallel()
		for _, request := range []*accounts_proto.ChangePasswordRequest{
			{
				Username: "invalid username",
				Password: "valid password",
			},
			{
				Username: "valid username",
				Password: "1234", // Invalid password: too short.
			},
			{
				Username: "",
				Password: "",
			},
		} {
			_, err = s.client.ChangePassword(ctx, request, creds)
			testutils.AssertErrorCode(s.T(), err, codes.InvalidArgument)
		}
	})
}

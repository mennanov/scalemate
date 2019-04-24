package server_test

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestTokenAuth() {
	ctx := context.Background()
	// Register a new user.
	registerRequest := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "email@mail.com",
		Password: "password",
	}
	registeredUser, err := s.client.Register(ctx, registerRequest)
	s.Require().NoError(err)

	// Get the auth tokens.
	authTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
		Request: &accounts_proto.PasswordAuthRequest_UserAuth{
			UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
				Username: registerRequest.Username,
				Password: registerRequest.Password,
			},
		},
	})
	s.Require().NoError(err)

	s.Run("succeeds", func() {
		s.T().Parallel()
		newAuthTokens, err := s.client.TokenAuth(ctx, &accounts_proto.TokenAuthRequest{
			RefreshToken: authTokens.RefreshToken,
		})
		s.Require().NoError(err)
		// Verify that these tokens can be used for subsequent RPCs.
		user, err := s.client.Get(ctx, &accounts_proto.UserLookupRequest{
			Request: &accounts_proto.UserLookupRequest_Id{
				Id: registeredUser.Id,
			},
		}, grpc.PerRPCCredentials(auth.NewSimpleJWTCredentials(newAuthTokens.AccessToken)))
		s.Require().NoError(err)
		s.Equal(registeredUser, user)
	})

	s.Run("invalid argument for invalid token", func() {
		s.T().Parallel()
		newAuthTokens, err := s.client.TokenAuth(ctx, &accounts_proto.TokenAuthRequest{
			RefreshToken: "invalid token",
		})
		testutils.AssertErrorCode(s.T(), err, codes.InvalidArgument)
		s.Nil(newAuthTokens)
	})
}

func (s *ServerTestSuite) TestTokenAuth_Banned() {
	ctx := context.Background()
	// Register a new user.
	registerRequest := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "email@mail.com",
		Password: "password",
	}
	registeredUser, err := s.client.Register(ctx, registerRequest)
	s.Require().NoError(err)
	// Get auth tokens before the user is banned.
	authTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
		Request: &accounts_proto.PasswordAuthRequest_UserAuth{
			UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
				Username: registerRequest.Username,
				Password: registerRequest.Password,
			},
		},
	})
	// Ban the user.
	s.db.MustExec("UPDATE users SET banned = true WHERE id = $1", registeredUser.Id)
	// Authenticate using the refresh token after the user is banned.
	newAuthTokens, err := s.client.TokenAuth(ctx, &accounts_proto.TokenAuthRequest{
		RefreshToken: authTokens.RefreshToken,
	})
	testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
	s.Nil(newAuthTokens)
}

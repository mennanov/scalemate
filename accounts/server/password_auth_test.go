package server_test

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestPasswordAuth() {
	ctx := context.Background()
	// Register a new user.
	registerRequest := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "email@mail.com",
		Password: "password",
	}
	registeredUser, err := s.client.Register(ctx, registerRequest)
	s.Require().NoError(err)

	s.Run("succeeds", func() {
		s.T().Parallel()
		authTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
			Request: &accounts_proto.PasswordAuthRequest_UserAuth{
				UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
					Username: registerRequest.Username,
					Password: registerRequest.Password,
				},
			},
		})
		s.Require().NoError(err)
		// Verify that these tokens can be used for subsequent RPCs.
		user, err := s.client.Get(ctx, &accounts_proto.UserLookupRequest{
			Request: &accounts_proto.UserLookupRequest_Id{
				Id: registeredUser.Id,
			},
		}, grpc.PerRPCCredentials(auth.NewSimpleJWTCredentials(authTokens.AccessToken)))
		s.Require().NoError(err)
		s.Equal(registeredUser, user)
	})

	s.Run("invalid argument for invalid password", func() {
		s.T().Parallel()
		authTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
			Request: &accounts_proto.PasswordAuthRequest_UserAuth{
				UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
					Username: registerRequest.Username,
					Password: "invalid password",
				},
			},
		})
		testutils.AssertErrorCode(s.T(), err, codes.InvalidArgument)
		s.Nil(authTokens)
	})

	s.Run("not found for invalid username", func() {
		s.T().Parallel()
		authTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
			Request: &accounts_proto.PasswordAuthRequest_UserAuth{
				UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
					Username: "invalid_username",
					Password: registerRequest.Password,
				},
			},
		})
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(authTokens)
	})
}

func (s *ServerTestSuite) TestPasswordAuth_Banned() {
	ctx := context.Background()
	// Register a new user.
	registerRequest := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "email@mail.com",
		Password: "password",
	}
	registeredUser, err := s.client.Register(ctx, registerRequest)
	s.Require().NoError(err)
	// Ban the user.
	s.db.MustExec("UPDATE users SET banned = true WHERE id = $1", registeredUser.Id)
	// Authenticate after the user is banned.
	authTokens, err := s.client.PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{
		Request: &accounts_proto.PasswordAuthRequest_UserAuth{
			UserAuth: &accounts_proto.PasswordAuthRequest_UserAuthRequest{
				Username: registerRequest.Username,
				Password: registerRequest.Password,
			},
		},
	})
	testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
	s.Nil(authTokens)
}

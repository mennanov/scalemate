package server_test

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestGet() {
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

	s.Run("succeeds for existing user", func() {
		s.T().Parallel()
		for _, lookupRequest := range []*accounts_proto.UserLookupRequest{
			{
				Request: &accounts_proto.UserLookupRequest_Id{
					Id: registeredUser.Id,
				},
			},
			{
				Request: &accounts_proto.UserLookupRequest_Username{
					Username: registeredUser.Username,
				},
			},
			{
				Request: &accounts_proto.UserLookupRequest_Email{
					Email: registeredUser.Email,
				},
			},
		} {
			user, err := s.client.Get(ctx, lookupRequest, creds)
			s.Require().NoError(err, lookupRequest)
			s.Equal(registeredUser, user)
		}
	})

	s.Run("not found for non-existing user", func() {
		s.T().Parallel()
		user, err := s.client.Get(ctx, &accounts_proto.UserLookupRequest{
			Request: &accounts_proto.UserLookupRequest_Id{
				Id: registeredUser.Id + 1,
			},
		}, creds)
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(user)
	})

	s.Run("permission denied for different user", func() {
		s.T().Parallel()
		// Register one more user.
		newUser, err := s.client.Register(ctx, &accounts_proto.RegisterRequest{
			Username: "different_username",
			Email:    "different_email@mail.com",
			Password: "password",
		})
		s.Require().NoError(err)
		// Get the recently registered user.
		user, err := s.client.Get(ctx, &accounts_proto.UserLookupRequest{
			Request: &accounts_proto.UserLookupRequest_Id{
				Id: newUser.Id,
			},
		}, creds)
		testutils.AssertErrorCode(s.T(), err, codes.PermissionDenied)
		s.Nil(user)
	})

	s.Run("unauthenticated without credentials", func() {
		s.T().Parallel()
		user, err := s.client.Get(ctx, &accounts_proto.UserLookupRequest{
			Request: &accounts_proto.UserLookupRequest_Id{
				Id: registeredUser.Id,
			},
		}) // No credentials provided.
		testutils.AssertErrorCode(s.T(), err, codes.Unauthenticated)
		s.Nil(user)
	})
}

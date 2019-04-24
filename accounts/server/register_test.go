package server_test

import (
	"context"
	"fmt"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ServerTestSuite) TestRegister() {
	ctx := context.Background()
	registerRequest := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "valid@email.com",
		Password: "password",
	}

	registeredUser, err := s.client.Register(ctx, registerRequest)
	s.Require().NoError(err)
	s.Equal(registerRequest.Username, registeredUser.Username)
	s.Equal(registerRequest.Email, registeredUser.Email)
	s.NotEqual(uint32(0), registeredUser.Id)

	// Verify that the event message is sent.
	s.NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_CREATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: registeredUser,
		},
	})))

	s.Run("duplicate registration fails", func() {
		s.T().Parallel()
		user, err := s.client.Register(ctx, registerRequest)
		testutils.AssertErrorCode(s.T(), err, codes.AlreadyExists)
		s.Nil(user)
	})

	s.Run("auth request succeeds for registered user", func() {
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
		s.NotNil(authTokens)
	})
}

func (s *ServerTestSuite) TestRegister_InvalidRequests() {
	ctx := context.Background()
	for i, registerRequest := range []*accounts_proto.RegisterRequest{
		{
			Username: "invalid username",
			Email:    "valid@email.com",
			Password: "valid password",
		},
		{
			Username: "aa", // Invalid username: too short.
			Email:    "valid@email.com",
			Password: "valid password",
		},
		{
			Username: "aaa",
			Email:    "valid@email.com",
			Password: "1234567", // Invalid password: too short.
		},
		{
			Username: "valid_username",
			Email:    "invalid email.com",
			Password: "valid password",
		},
		{
			Username: "valid_username",
			Email:    "valid@email.com",
			Password: "", // Invalid (empty) password.
		},
		{
			Username: "", // Empty username.
			Email:    "valid@email.com",
			Password: "valid password",
		},
		{
			Username: "valid_username", // Empty username.
			Email:    "",               // Empty email.
			Password: "valid password",
		},
		{
			Username: "",
			Email:    "",
			Password: "",
		},
	} {
		// Capture the request variable.
		registerRequest := registerRequest
		s.Run(fmt.Sprintf("invalid request %d", i), func() {
			s.T().Parallel()
			user, err := s.client.Register(ctx, registerRequest)
			testutils.AssertErrorCode(s.T(), err, codes.InvalidArgument)
			s.Nil(user)
		})

	}
}

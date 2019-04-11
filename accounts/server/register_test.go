package server_test

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *ServerTestSuite) TestRegister() {
	ctx := context.Background()
	req := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "valid@email.com",
		Password: "password",
	}

	_, err := s.client.Register(ctx, req)
	s.Require().NoError(err)
	s.NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_CREATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{},
		},
	})))

	// Verify that the user is created in DB.
	user := &models.User{}
	err = s.db.Where("username = ?", req.Username).First(user).Error
	s.Require().NoError(err)
	s.NotEqual("", user.PasswordHash)
}

func (s *ServerTestSuite) TestRegisterDuplicates() {
	existingUser := s.createTestUserQuick("password")

	testCases := []accounts_proto.RegisterRequest{
		{
			Username: existingUser.Username,
			Email:    existingUser.Email,
			Password: "password",
		},
		{
			Username: existingUser.Username,
			Email:    "unique@email.com",
			Password: "password",
		},
		{
			Username: "unique_username",
			Email:    existingUser.Email,
			Password: "password",
		},
	}

	var count int
	user := &models.User{}

	for _, testCase := range testCases {
		ctx := context.Background()

		_, err := s.client.Register(ctx, &testCase)

		s.assertGRPCError(err, codes.AlreadyExists)

		s.db.Model(user).Count(&count)
		// Only 1 existing user is expected.
		s.Equal(1, count)
	}
}

func (s *ServerTestSuite) TestRegisterValidationFails() {
	testCases := []accounts_proto.RegisterRequest{
		{
			Username: "invalid username",
			Email:    "valid@email.com",
			Password: "password",
		},
		{
			Username: "valid_username",
			Email:    "invalid email",
			Password: "password",
		},
		{
			Username: "valid_username",
			Email:    "",
			Password: "password",
		},
	}

	for _, testCase := range testCases {
		ctx := context.Background()

		_, err := s.client.Register(ctx, &testCase)

		s.assertGRPCError(err, codes.InvalidArgument)
	}

	var count int
	s.db.Model(&models.User{}).Count(&count)
	// No users should be created.
	s.Equal(0, count)
}

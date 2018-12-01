package server_test

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestRegister() {
	messages, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.AccountsAMQPExchangeName)
	s.Require().NoError(err)

	ctx := context.Background()
	req := &accounts_proto.RegisterRequest{
		Username: "username",
		Email:    "valid@email.com",
		Password: "password",
	}

	_, err = s.client.Register(ctx, req)
	s.Require().NoError(err)

	// Verify that user is created in DB.
	user := &models.User{}
	err = s.service.DB.Where("username = ?", req.Username).First(user).Error
	s.Require().NoError(err)
	s.NotEqual("", user.PasswordHash)
	utils.WaitForMessages(messages, "accounts.user.created")
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

		s.service.DB.Model(user).Count(&count)
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
	s.service.DB.Model(&models.User{}).Count(&count)
	// No users should be created.
	s.Equal(0, count)
}

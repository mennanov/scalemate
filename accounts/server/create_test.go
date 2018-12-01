package server_test

import (
	"context"
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestCreate() {
	messages, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.AccountsAMQPExchangeName)
	s.Require().NoError(err)

	ctx := context.Background()
	req := &accounts_proto.CreateUserRequest{
		User: &accounts_proto.User{
			Username: "username",
			Email:    "valid@email.com",
		},
		Password: "password",
	}

	res, err := s.client.Create(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Require().NoError(err)

	s.NotEqual(0, res.GetId())
	expected := map[string]interface{}{
		"Username": req.User.Username,
		"Email":    req.User.Email,
		"Role":     accounts_proto.User_UNKNOWN,
		"Banned":   req.User.Banned,
	}
	mask := fieldmask_utils.MaskFromString("username,email,role,banned")
	actual := make(map[string]interface{})
	err = fieldmask_utils.StructToMap(mask, res, actual, generator.CamelCase, stringEye)
	s.Require().NoError(err)
	s.Equal(expected, actual)

	s.NotNil(res.GetId())
	s.NotNil(res.GetCreatedAt())
	s.NotNil(res.GetUpdatedAt())

	// Verify that user is created in DB.
	user := &models.User{}
	err = s.service.DB.First(user, res.GetId()).Error
	s.Require().NoError(err)
	s.NotEqual("", user.PasswordHash)
	utils.WaitForMessages(messages, "accounts.user.created")
}

func (s *ServerTestSuite) TestCreateDuplicates() {
	messages, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.AccountsAMQPExchangeName)
	s.Require().NoError(err)

	ctx := context.Background()
	req := &accounts_proto.CreateUserRequest{
		User: &accounts_proto.User{
			Username: "username",
			Email:    "valid@email.com",
		},
		Password: "password",
	}

	count := 0
	creds := s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN)

	_, err = s.client.Create(ctx, req, creds)
	s.Require().NoError(err)

	users := &[]models.User{}

	s.service.DB.Find(&users).Count(&count)
	s.Equal(1, count)

	// The second consecutive request should fail.
	_, err = s.client.Create(ctx, req, creds)
	s.assertGRPCError(err, codes.AlreadyExists)

	s.service.DB.Find(&users).Count(&count)
	s.Equal(1, count)
	utils.WaitForMessages(messages, "accounts.user.created")
}

func (s *ServerTestSuite) TestCreateValidationFails() {
	ctx := context.Background()
	req := &accounts_proto.CreateUserRequest{
		User: &accounts_proto.User{
			Username: "u", // invalid username: too short
			Email:    "invalid email",
		},
		Password: "p", // invalid password: too short
	}

	_, err := s.client.Create(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.assertGRPCError(err, codes.InvalidArgument)
}

func (s *ServerTestSuite) TestCreateUnauthenticated() {
	ctx := context.Background()
	req := &accounts_proto.CreateUserRequest{}

	// Make a request with no access credentials.
	_, err := s.client.Create(ctx, req)
	s.assertGRPCError(err, codes.Unauthenticated)
}

func (s *ServerTestSuite) TestCreatePermissionDenied() {
	ctx := context.Background()
	req := &accounts_proto.CreateUserRequest{}

	// Make a request with insufficient access credentials.
	_, err := s.client.Create(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_USER))
	s.assertGRPCError(err, codes.PermissionDenied)
}

func stringEye(s string) string {
	return s
}

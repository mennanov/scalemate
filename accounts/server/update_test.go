package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestUpdate() {
	messages, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.AccountsAMQPExchangeName)
	s.Require().NoError(err)

	user := s.createTestUser(&models.User{
		Username: "username",
		Email:    "email@mail.com",
		Banned:   true,
		Role:     accounts_proto.User_ADMIN,
	}, "password")

	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{
		Lookup: &accounts_proto.UserLookupRequest{Id: uint32(user.ID)},
		User:   &accounts_proto.User{Username: "new_username", Role: accounts_proto.User_USER, Banned: false},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"role", "banned", "username"},
		},
	}

	res, err := s.client.Update(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Require().NoError(err)
	s.NotNil(res)

	// Verify that the user is updated.
	s.Equal(res.Username, req.User.Username)
	s.Equal(res.Role, req.User.Role)
	s.Equal(res.Banned, req.User.Banned)
	// Check messages.
	s.NoError(utils.ExpectMessages(messages, time.Millisecond*50, "accounts.user.updated"))
}

func (s *ServerTestSuite) TestUpdateByFieldMaskOnly() {
	messages, err := utils.SetUpAMQPTestConsumer(s.service.AMQPConnection, utils.AccountsAMQPExchangeName)
	s.Require().NoError(err)
	user := s.createTestUser(&models.User{
		Username: "username",
		Email:    "email@mail.com",
		Banned:   true,
		Role:     accounts_proto.User_ADMIN,
	}, "password")

	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{
		Lookup: &accounts_proto.UserLookupRequest{Id: uint32(user.ID)},
		User:   &accounts_proto.User{Username: "new_username", Role: accounts_proto.User_USER, Banned: false},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"role"},
		},
	}

	res, err := s.client.Update(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Require().NoError(err)
	s.NotNil(res)

	// Verify that only "role" of the user is updated.
	s.Equal(user.Username, res.Username)
	s.Equal(req.User.Role, res.Role)
	s.Equal(user.Banned, res.Banned)
	s.NoError(utils.ExpectMessages(messages, time.Millisecond*50, `accounts.user.updated\..*?role.*?`))
}

func (s *ServerTestSuite) TestUpdateByEmptyFieldMask() {
	user := s.createTestUser(&models.User{
		Username: "username",
		Email:    "email@mail.com",
		Banned:   true,
		Role:     accounts_proto.User_ADMIN,
	}, "password")

	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{
		Lookup:     &accounts_proto.UserLookupRequest{Id: uint32(user.ID)},
		User:       &accounts_proto.User{Username: "new_username", Role: accounts_proto.User_USER, Banned: false},
		UpdateMask: &field_mask.FieldMask{},
	}

	res, err := s.client.Update(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Nil(res)
	s.assertGRPCError(err, codes.InvalidArgument)
}

func (s *ServerTestSuite) TestUpdateLookupFails() {
	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{
		Lookup: &accounts_proto.UserLookupRequest{Id: uint32(1)},
		User:   &accounts_proto.User{Username: "new_username"},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"username"},
		},
	}

	res, err := s.client.Update(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Nil(res)
	s.assertGRPCError(err, codes.NotFound)
}

func (s *ServerTestSuite) TestUpdateUnauthenticated() {
	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{}

	// Make a request with no access credentials.
	_, err := s.client.Update(ctx, req)
	s.assertGRPCError(err, codes.Unauthenticated)
}

func (s *ServerTestSuite) TestUpdatePermissionDenied() {
	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{}

	// Make a request with insufficient access credentials.
	_, err := s.client.Update(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_USER))
	s.assertGRPCError(err, codes.PermissionDenied)
}

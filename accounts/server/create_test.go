package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *ServerTestSuite) TestCreate() {
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
	s.Require().NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_CREATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{},
		},
	})))

	s.NotEqual(uint64(0), res.GetId())
	s.Equal(req.User.Username, res.Username)
	s.Equal(req.User.Email, res.Email)
	s.Equal(accounts_proto.User_UNKNOWN, res.Role)
	s.False(res.Banned)
	s.NotNil(res.GetCreatedAt())
	s.Nil(res.GetUpdatedAt())

	// Verify that user is created in DB.
	user := &models.User{}
	s.Require().NoError(user.LookUp(s.db, &accounts_proto.UserLookupRequest{Id: res.Id}))
	s.NotEqual("", user.PasswordHash)
}

func (s *ServerTestSuite) TestCreate_FailsForDuplicates() {
	ctx := context.Background()
	req := &accounts_proto.CreateUserRequest{
		User: &accounts_proto.User{
			Username: "username",
			Email:    "valid@email.com",
		},
		Password: "password",
	}

	creds := s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN)

	_, err := s.client.Create(ctx, req, creds)
	s.Require().NoError(err)

	users := &[]models.User{}

	var count int
	s.db.Find(&users).Count(&count)
	s.Equal(1, count)

	// The second consecutive request should fail.
	_, err = s.client.Create(ctx, req, creds)
	s.assertGRPCError(err, codes.AlreadyExists)

	s.db.Find(&users).Count(&count)
	s.Equal(1, count)
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

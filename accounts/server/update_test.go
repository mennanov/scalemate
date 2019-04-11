package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *ServerTestSuite) TestUpdate() {
	user := s.createTestUser(&models.User{
		Username: "username",
		Email:    "email@mail.com",
		Banned:   true,
		Role:     accounts_proto.User_ADMIN,
	}, "password")

	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{
		Lookup: &accounts_proto.UserLookupRequest{Id: user.ID},
		User:   &accounts_proto.User{Username: "new_username", Role: accounts_proto.User_USER, Banned: false},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"role", "banned", "username"},
		},
	}

	res, err := s.client.Update(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Require().NoError(err)
	s.NotNil(res)
	s.NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_UPDATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{Id: user.ID},
		},
	})))

	// Verify that the user is updated.
	s.Equal(res.Username, req.User.Username)
	s.Equal(res.Role, req.User.Role)
	s.Equal(res.Banned, req.User.Banned)
}

func (s *ServerTestSuite) TestUpdateByFieldMaskOnly() {
	user := s.createTestUser(&models.User{
		Username: "username",
		Email:    "email@mail.com",
		Banned:   true,
		Role:     accounts_proto.User_ADMIN,
	}, "password")

	ctx := context.Background()
	req := &accounts_proto.UpdateUserRequest{
		Lookup: &accounts_proto.UserLookupRequest{Id: user.ID},
		User:   &accounts_proto.User{Username: "new_username", Role: accounts_proto.User_USER, Banned: false},
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"role"},
		},
	}

	res, err := s.client.Update(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Require().NoError(err)
	s.NotNil(res)
	s.NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_UPDATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{Id: user.ID},
		},
	})))

	// Verify that only "role" of the user is updated.
	s.Equal(user.Username, res.Username)
	s.Equal(req.User.Role, res.Role)
	s.Equal(user.Banned, res.Banned)
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
		Lookup:     &accounts_proto.UserLookupRequest{Id: user.ID},
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
		Lookup: &accounts_proto.UserLookupRequest{Id: 1},
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

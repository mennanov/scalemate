package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events"
)

func (s *ServerTestSuite) TestDelete() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{Id: user.ID}

	res, err := s.client.Delete(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Require().NoError(err)
	s.NotNil(res)
	s.NoError(s.messagesHandler.ExpectMessages(events.KeyForEvent(&events_proto.Event{
		Type:    events_proto.Event_DELETED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{Id: user.ID},
		},
	})))

	// Verify that the user is deleted.
	err = user.LookUp(s.db, &accounts_proto.UserLookupRequest{Id: user.ID})
	s.assertGRPCError(errors.Cause(err), codes.NotFound)
}

func (s *ServerTestSuite) TestDeleteLookupFails() {
	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{Id: 1}

	_, err := s.client.Delete(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.assertGRPCError(err, codes.NotFound)
}

func (s *ServerTestSuite) TestDeleteUnauthenticated() {
	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{}

	// Make a request with no access credentials.
	_, err := s.client.Delete(ctx, req)
	s.assertGRPCError(err, codes.Unauthenticated)
}

func (s *ServerTestSuite) TestDeletePermissionDenied() {
	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{}

	// Make a request with insufficient access credentials.
	_, err := s.client.Delete(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_USER))
	s.assertGRPCError(err, codes.PermissionDenied)
}

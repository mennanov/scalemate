package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestDelete() {
	messages, err := events.NewAMQPRawConsumer(s.amqpChannel, events.AccountsAMQPExchangeName, "", "#")
	s.Require().NoError(err)

	user := s.createTestUserQuick("password")

	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{Id: uint32(user.ID)}

	res, err := s.client.Delete(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Nil(err)
	s.NotNil(res)

	// Verify that the user is deleted.
	err = user.LookUp(s.db, &accounts_proto.UserLookupRequest{Id: uint32(user.ID)})
	s.assertGRPCError(errors.Cause(err), codes.NotFound)
	utils.WaitForMessages(messages, "accounts.user.deleted")
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

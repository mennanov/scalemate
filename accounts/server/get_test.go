package server_test

import (
	"context"
	"time"

	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"
)

func (s *ServerTestSuite) TestGet() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{Id: uint32(user.ID)}

	res, err := s.client.Get(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.Require().NoError(err)
	s.Require().NotNil(res)

	expected := map[string]interface{}{
		"Id":       uint64(user.ID),
		"Username": user.Username,
		"Email":    user.Email,
		"Role":     accounts_proto.User_USER,
		"Banned":   user.Banned,
	}
	mask := fieldmask_utils.MaskFromString("Id,Username,Email,Role,Banned")
	actual := make(map[string]interface{})
	err = fieldmask_utils.StructToMap(mask, res, actual)
	s.Require().NoError(err)
	s.Equal(expected, actual)
}

func (s *ServerTestSuite) TestGetLookupFails() {
	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{Id: 1}

	_, err := s.client.Get(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_ADMIN))
	s.assertGRPCError(err, codes.NotFound)
}

func (s *ServerTestSuite) TestGetUnauthenticated() {
	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{}

	// Make a request with no access credentials.
	_, err := s.client.Get(ctx, req)
	s.assertGRPCError(err, codes.Unauthenticated)
}

func (s *ServerTestSuite) TestGetPermissionDenied() {
	ctx := context.Background()
	req := &accounts_proto.UserLookupRequest{}

	// Make a request with insufficient access credentials.
	_, err := s.client.Get(ctx, req, s.accessCredentialsQuick(time.Minute, accounts_proto.User_USER))
	s.assertGRPCError(err, codes.PermissionDenied)
}

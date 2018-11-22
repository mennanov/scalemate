package server_test

import (
	"context"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/accounts/models"
	"google.golang.org/grpc/codes"
)

func (s *ServerTestSuite) TestTokenAuth() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()

	// Obtain auth tokens first.
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res, user, time.Now(), "")

	req2 := &accounts_proto.TokenAuthRequest{
		RefreshToken: res.GetRefreshToken(),
	}

	res2, err := s.client.TokenAuth(ctx, req2)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res2, user, time.Now(), "")

	s.NotEqual(res.AccessToken, res2.AccessToken)
	s.NotEqual(res.RefreshToken, res2.RefreshToken)
}

func (s *ServerTestSuite) TestTokenAuth_WithNodeName() {
	user := s.createTestUserQuick("password")

	node := &models.Node{
		Username:    user.Username,
		Name:        "node_name",
		CpuModel:    "Intel Core i7 @ 2.20GHz",
		GpuModel:    "Intel Iris Pro 1536MB",
		MemoryModel: "DDR3-1600MHz",
		DiskModel:   "251GB APPLE SSD SM0256F",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	ctx := context.Background()
	req := &accounts_proto.PasswordNodeAuthRequest{
		Username:    node.Username,
		Password:    "password",
		NodeName:    node.Name,
		CpuModel:    node.CpuModel,
		GpuModel:    node.GpuModel,
		MemoryModel: node.MemoryModel,
		DiskModel:   node.DiskModel,
	}

	res, err := s.client.PasswordNodeAuth(ctx, req)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res, user, time.Now(), node.Name)

	req2 := &accounts_proto.TokenAuthRequest{
		RefreshToken: res.GetRefreshToken(),
	}

	res2, err := s.client.TokenAuth(ctx, req2)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res2, user, time.Now(), node.Name)

	s.NotEqual(res.AccessToken, res2.AccessToken)
	s.NotEqual(res.RefreshToken, res2.RefreshToken)
}

func (s *ServerTestSuite) TestTokenAuthInvalidTokenFormat() {
	s.createTestUserQuick("password")

	ctx := context.Background()

	req := &accounts_proto.TokenAuthRequest{
		RefreshToken: "invalid token",
	}

	res, err := s.client.TokenAuth(ctx, req)
	s.Nil(res)
	s.assertGRPCError(err, codes.InvalidArgument)
}

func (s *ServerTestSuite) TestTokenAuthBannedUser() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()

	// Obtain auth tokens first for a valid (not banned) user.
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res, user, time.Now(), "")

	// Mark user as banned.
	user.Banned = true
	s.service.DB.Save(user)

	req2 := &accounts_proto.TokenAuthRequest{
		RefreshToken: res.GetRefreshToken(),
	}

	res2, err := s.client.TokenAuth(ctx, req2)
	s.Nil(res2)
	s.assertGRPCError(err, codes.PermissionDenied)
}

func (s *ServerTestSuite) TestTokenAuthUseAccessTokenInsteadOfRefresh() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()

	// Obtain auth tokens first.
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res, user, time.Now(), "")

	req2 := &accounts_proto.TokenAuthRequest{
		RefreshToken: res.GetAccessToken(),
	}

	res2, err := s.client.TokenAuth(ctx, req2)
	s.Nil(res2)
	s.assertGRPCError(err, codes.InvalidArgument)
}

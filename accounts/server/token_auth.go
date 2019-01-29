package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/auth"
)

// TokenAuth performs authentication by a JWT refresh token.
// If all credentials are valid a new pair of access and refresh JWT are issued.
func (s AccountsServer) TokenAuth(
	ctx context.Context,
	r *accounts_proto.TokenAuthRequest,
) (*accounts_proto.AuthTokens, error) {
	claims, err := auth.NewClaimsFromStringVerified(r.GetRefreshToken(), s.JWTSecretKey)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != auth.TokenTypeRefresh {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token type is expected", claims.TokenType)
	}

	user := &models.User{}

	if err := user.LookUp(s.DB, &accounts_proto.UserLookupRequest{Username: claims.Username}); err != nil {
		return nil, err
	}

	if err := user.IsAllowedToAuthenticate(); err != nil {
		return nil, err
	}

	// Generate auth tokens.
	response, err := user.
		GenerateAuthTokensResponse(s.AccessTokenTTL, s.RefreshTokenTTL, s.JWTSecretKey, claims.NodeName)
	if err != nil {
		return nil, err
	}

	return response, nil
}

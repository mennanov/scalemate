package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TokenAuth performs authentication by a JWT refresh token.
// If all credentials are valid a new pair of access and refresh JWT are issued.
func (s AccountsServer) TokenAuth(ctx context.Context, r *accounts_proto.TokenAuthRequest) (*accounts_proto.AuthTokens, error) {
	claims, err := auth.NewClaimsFromStringVerified(r.GetRefreshToken(), s.JWTSecretKey)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, status.Errorf(codes.InvalidArgument, "token of type 'refresh' is expected, got: %s",
			claims.TokenType)
	}

	user := &models.User{}

	if err := user.LookUp(s.DB, &accounts_proto.UserLookupRequest{Username: claims.Username}); err != nil {
		return nil, err
	}

	if err := user.IsAllowedToAuthenticate(); err != nil {
		return nil, err
	}

	// Generate auth tokens.
	response, err := user.GenerateAuthTokensResponse(s.AccessTokenTTL, s.RefreshTokenTTL, s.JWTSecretKey, claims.NodeName)
	if err != nil {
		return nil, err
	}

	return response, nil
}

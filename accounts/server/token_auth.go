package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
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
	claims, err := auth.NewClaimsFromStringVerified(r.GetRefreshToken(), s.jwtSecretKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse JWT")
	}

	if claims.TokenType != auth.TokenTypeRefresh {
		return nil, status.Errorf(codes.InvalidArgument, "refresh token type is expected", claims.TokenType)
	}

	user, err := models.UserLookUp(s.db, &accounts_proto.UserLookupRequest{
		Request: &accounts_proto.UserLookupRequest_Username{Username: claims.Username},
	})
	if err != nil {
		return nil, errors.Wrap(err, "UserLookUp failed")
	}

	if err := user.IsAllowedToAuthenticate(); err != nil {
		return nil, err
	}

	// Generate auth tokens.
	response, err := user.GenerateAuthTokens(s.accessTokenTTL, s.refreshTokenTTL, s.jwtSecretKey, claims.NodeName)
	if err != nil {
		return nil, errors.Wrap(err, "GenerateAuthTokens failed")
	}

	return response, nil
}

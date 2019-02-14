package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
)

// PasswordAuth authenticates a user by a username and a password.
// It creates a JWT to be used later for other methods across all the other services.
func (s AccountsServer) PasswordAuth(
	ctx context.Context,
	r *accounts_proto.PasswordAuthRequest,
) (*accounts_proto.AuthTokens, error) {
	user := &models.User{}

	if err := user.LookUp(s.db, &accounts_proto.UserLookupRequest{Username: r.GetUsername()}); err != nil {
		return nil, err
	}

	if err := user.ComparePassword(r.GetPassword()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "incorrect password")
	}

	// Perform this check after the password is verified to avoid banned users scans.
	if err := user.IsAllowedToAuthenticate(); err != nil {
		return nil, err
	}

	// Generate auth tokens.
	response, err := user.GenerateAuthTokensResponse(s.accessTokenTTL, s.refreshTokenTTL, s.jwtSecretKey, "")
	if err != nil {
		return nil, err
	}

	return response, nil
}

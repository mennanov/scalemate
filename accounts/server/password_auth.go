package server

import (
	"bytes"
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
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
	var username, password, nodeName string
	var fingerprint []byte
	switch req := r.Request.(type) {
	case *accounts_proto.PasswordAuthRequest_UserAuth:
		username, password = req.UserAuth.Username, req.UserAuth.Password

	case *accounts_proto.PasswordAuthRequest_NodeAuth:
		username, password, nodeName = req.NodeAuth.Username, req.NodeAuth.Password, req.NodeAuth.NodeName
		fingerprint = req.NodeAuth.NodeFingerprint
	}
	user, err := models.UserLookUp(s.db, &accounts_proto.UserLookupRequest{
		Request: &accounts_proto.UserLookupRequest_Username{
			Username: username,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "UserLookUp failed")
	}

	if err := user.ComparePassword(password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "incorrect password")
	}

	// Perform this check after the password is verified to avoid banned users scans.
	if err := user.IsAllowedToAuthenticate(); err != nil {
		return nil, err
	}

	if nodeName != "" {
		node, err := models.NodeLookUp(s.db, username, nodeName)
		if err != nil {
			return nil, errors.Wrap(err, "NodeLookUp failed")
		}
		if !bytes.Equal(node.Fingerprint, fingerprint) {
			return nil, status.Error(codes.InvalidArgument, "fingerprint is incorrect")
		}
	}

	// Generate auth tokens.
	response, err := user.GenerateAuthTokens(s.accessTokenTTL, s.refreshTokenTTL, s.jwtSecretKey, nodeName)
	if err != nil {
		return nil, errors.Wrap(err, "GenerateAuthTokens failed")
	}

	return response, nil
}

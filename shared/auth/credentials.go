package auth

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// TokensSaver is a function that persists the given AuthTokens so that they can be reused later.
type TokensSaver func(*accounts_proto.AuthTokens) error

// JWTCredentials represents the JWT gRPC credentials.
type JWTCredentials struct {
	save   TokensSaver
	client accounts_proto.AccountsClient
	tokens *accounts_proto.AuthTokens
}

// NewJWTCredentials creates a new instance of JWTCredentials.
func NewJWTCredentials(client accounts_proto.AccountsClient, tokens *accounts_proto.AuthTokens, tokensSaver TokensSaver) *JWTCredentials {
	return &JWTCredentials{
		client: client,
		save:   tokensSaver,
		tokens: tokens,
	}
}

// GetRequestMetadata returns authorization headers with JWT access token.
// If the access token is expired, then the new one is obtained using the refresh token.
func (j *JWTCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	claimsAccess, err := NewClaimsFromString(j.tokens.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse access token")
	}

	if err := claimsAccess.Valid(); err != nil {
		// Obtain a new access token using the refresh token.
		logrus.Debugf("Access token is invalid: %s", err)

		if _, err := NewClaimsFromString(j.tokens.RefreshToken); err != nil {
			return nil, errors.Wrap(err, "failed to parse refresh token")
		}

		refreshRequest := &accounts_proto.TokenAuthRequest{RefreshToken: j.tokens.RefreshToken}
		logrus.WithFields(logrus.Fields{
			"RefreshToken": j.tokens.RefreshToken,
		}).Debugf("Requesting a new tokens pair using refresh token")

		tokens, err := j.client.TokenAuth(ctx, refreshRequest)
		if err != nil {
			return nil, errors.Wrap(err, "failed to authenticate using refresh token. Please, login.")
		}

		// Save refreshed tokens.
		if err := j.save(tokens); err != nil {
			return nil, errors.Wrap(err, "failed to save new tokens pair")
		}
		j.tokens = tokens
	}

	return map[string]string{
		"authorization": "Bearer " + j.tokens.AccessToken,
	}, nil
}

// RequireTransportSecurity requires TLS for gRPC connection.
func (j *JWTCredentials) RequireTransportSecurity() bool {
	return true
}

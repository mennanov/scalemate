package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mennanov/scalemate/shared/auth"
)

func createToken(ttl time.Duration, tokenType string) string {
	now := time.Now()
	expiresAt := now.Add(ttl).Unix()

	claims := &auth.Claims{
		Username:  "username",
		Role:      accounts_proto.User_ADMIN,
		TokenType: tokenType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
			Issuer:    "Scalemate.io",
			IssuedAt:  now.Unix(),
			Id:        uuid.New().String(),
		},
	}
	secret := []byte("allyourbase")
	tokenString, err := claims.SignedString(secret)
	if err != nil {
		panic(err)
	}
	return tokenString
}

func TestGetRequestMetadataValidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := NewMockAccountsClient(ctrl)
	// gRPC call is not expected if the access token is valid.
	client.EXPECT().TokenAuth(gomock.Any(), gomock.Any()).Times(0)

	saverCalled := false
	tokenSaver := func(*accounts_proto.AuthTokens) error {
		saverCalled = true
		return nil
	}
	authTokens := &accounts_proto.AuthTokens{
		AccessToken:  createToken(time.Minute, "access"),
		RefreshToken: createToken(30*time.Minute, "refresh"),
	}
	creds := auth.NewJWTCredentials(client, authTokens, tokenSaver)
	metadata, err := creds.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"authorization": "Bearer " + authTokens.AccessToken}, metadata)
	assert.False(t, saverCalled)
}

func TestGetRequestMetadataInvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Existing AuthTokens.
	authTokens := &accounts_proto.AuthTokens{
		AccessToken:  createToken(-time.Minute, "access"), // Token expired 1 minute ago.
		RefreshToken: createToken(5*time.Minute, "refresh"),
	}

	// AuthTokens to be returned by the client mock.
	newAuthTokens := &accounts_proto.AuthTokens{
		AccessToken:  createToken(time.Minute, "access"),
		RefreshToken: createToken(30*time.Minute, "refresh"),
	}

	// TokenAuthRequest to be used to refresh tokens.
	tokenAuthRequest := &accounts_proto.TokenAuthRequest{
		RefreshToken: authTokens.RefreshToken,
	}

	client := NewMockAccountsClient(ctrl)

	// gRPC call is expected since the access token is expired.
	client.EXPECT().TokenAuth(ctx, tokenAuthRequest).Return(newAuthTokens, nil)

	saverCalled := false
	tokenSaver := func(*accounts_proto.AuthTokens) error {
		saverCalled = true
		return nil
	}

	creds := auth.NewJWTCredentials(client, authTokens, tokenSaver)
	metadata, err := creds.GetRequestMetadata(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"authorization": "Bearer " + newAuthTokens.AccessToken}, metadata)
	assert.True(t, saverCalled)
}

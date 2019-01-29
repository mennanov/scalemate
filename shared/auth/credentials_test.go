package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

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
		AccessToken:  utils.CreateTestingTokenString(time.Minute, auth.TokenTypeAccess, ""),
		RefreshToken: utils.CreateTestingTokenString(30*time.Minute, auth.TokenTypeRefresh, ""),
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
		AccessToken:  utils.CreateTestingTokenString(-time.Minute, auth.TokenTypeAccess, ""), // Token expired 1 minute ago.
		RefreshToken: utils.CreateTestingTokenString(5*time.Minute, auth.TokenTypeRefresh, ""),
	}

	// AuthTokens to be returned by the client mock.
	newAuthTokens := &accounts_proto.AuthTokens{
		AccessToken:  utils.CreateTestingTokenString(time.Minute, auth.TokenTypeAccess, ""),
		RefreshToken: utils.CreateTestingTokenString(30*time.Minute, auth.TokenTypeRefresh, ""),
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

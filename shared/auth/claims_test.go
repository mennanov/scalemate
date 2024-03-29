package auth_test

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mennanov/scalemate/shared/auth"
)

func TestClaims_SignedString(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Minute).Unix()

	claims := &auth.Claims{
		Username:  "username",
		TokenType: auth.TokenTypeAccess,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
			Issuer:    "Scalemate.io",
			IssuedAt:  now.Unix(),
			Id:        uuid.New().String(),
		},
	}
	secret := []byte("allyourbase")

	tokenString, err := claims.SignedString(secret)

	require.NoError(t, err)
	assert.NotEqual(t, "", tokenString)

	t.Run("NewClaimsFromStringVerified", func(t *testing.T) {
		t.Parallel()
		claimsFromToken, err := auth.NewClaimsFromStringVerified(tokenString, secret)
		require.NoError(t, err)
		assert.Equal(t, claims.Username, claimsFromToken.Username)
		assert.Equal(t, claims.TokenType, claimsFromToken.TokenType)
		assert.Equal(t, claims.ExpiresAt, claimsFromToken.ExpiresAt)
		assert.Equal(t, claims.Id, claimsFromToken.Id)
	})

	t.Run("NewClaimsFromStringVerifiedInvalidSecret", func(t *testing.T) {
		t.Parallel()
		_, err := auth.NewClaimsFromStringVerified(tokenString, []byte("invalid secret key"))
		require.Error(t, err)
	})

	t.Run("NewClaimsFromString", func(t *testing.T) {
		t.Parallel()
		claimsFromToken, err := auth.NewClaimsFromString(tokenString)
		require.NoError(t, err)
		assert.Equal(t, claims.Username, claimsFromToken.Username)
		assert.Equal(t, claims.TokenType, claimsFromToken.TokenType)
		assert.Equal(t, claims.ExpiresAt, claimsFromToken.ExpiresAt)
		assert.Equal(t, claims.Id, claimsFromToken.Id)
	})
}

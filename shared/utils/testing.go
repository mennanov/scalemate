package utils

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
)

// CreateTestingTokenString creates a JWT testing string for a given ttl and token type.
func CreateTestingTokenString(ttl time.Duration, tokenType auth.TokenType, username string) string {
	now := time.Now()
	expiresAt := now.Add(ttl).Unix()

	if username == "" {
		username = "username"
	}

	claims := &auth.Claims{
		Username:  username,
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

// CreateAndSaveTestingTokens saves auto-generated auth tokens for a given username and returns a function that deletes
// them.
func CreateAndSaveTestingTokens(t *testing.T, username string) func() {
	authTokens := &accounts_proto.AuthTokens{
		AccessToken:  CreateTestingTokenString(time.Minute, auth.TokenTypeAccess, username),
		RefreshToken: CreateTestingTokenString(30*time.Minute, auth.TokenTypeRefresh, username),
	}
	err := auth.SaveTokens(authTokens)
	require.NoError(t, err)
	return func() {
		require.NoError(t, auth.DeleteTokens())
	}
}

// GetAllErrors returns errors with all possible error codes and an additional unknown error.
func GetAllErrors() []error {
	errs := make([]error, 18)
	for i := 0; i <= 16; i++ {
		// Start with non zero errors.
		errs[i] = status.Error(codes.Code(i+1), "Error message")
	}
	// Add an unknown error.
	errs[17] = errors.New("Unknown error")
	return errs
}

// CreateTestingDatabase creates a new testing database and returns a connection to it.
// It drops the existing database with the given name before creating it.
func CreateTestingDatabase(dbURL, dbName string) (*gorm.DB, error) {
	db, err := ConnectDBFromEnv(dbURL)
	if err != nil {
		return nil, errors.Wrap(err, "ConnectDBFromEnv failed")
	}
	if err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)).Error; err != nil {
		return nil, errors.Wrapf(err, "failed to drop a database %s", dbName)
	}
	if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)).Error; err != nil {
		return nil, errors.Wrapf(err, "failed to create a database %s", dbName)
	}

	newUrl := regexp.MustCompile(`dbname=(\w+)`).ReplaceAllString(dbURL, fmt.Sprintf("dbname=%s", dbName))
	return ConnectDBFromEnv(newUrl)
}

// Package testutils provides convenient functions for using in tests.
package testutils

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
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

// AssertErrorCode asserts that a given gRPC error has a specific code.
func AssertErrorCode(t *testing.T, err error, code codes.Code) {
	require.Error(t, err)
	statusCode, ok := status.FromError(errors.Cause(err))
	require.True(t, ok, "Not a status error")
	assert.Equal(t, code, statusCode.Code())
}

// CreateTestingDatabase creates a new testing database and returns a connection to it.
// It drops the existing database with the given name before creating it.
func CreateTestingDatabase(dbURL, dbName string) (*sqlx.DB, error) {
	db, err := utils.ConnectDBFromEnv(dbURL)
	if err != nil {
		return nil, errors.Wrap(err, "ConnectDBFromEnv failed")
	}
	if _, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)); err != nil {
		return nil, errors.Wrapf(err, "failed to drop a database %s", dbName)
	}
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		return nil, errors.Wrapf(err, "failed to create a database %s", dbName)
	}

	newURL := regexp.MustCompile(`dbname=(\w+)`).ReplaceAllString(dbURL, fmt.Sprintf("dbname=%s", dbName))
	return utils.ConnectDBFromEnv(newURL)
}

// TruncateTables truncates the database tables and resets associated sequence generators.
func TruncateTables(db sqlx.Ext) {
	rows, err := db.Queryx("SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public'")
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			panic(err)
		}
		if _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName)); err != nil {
			panic(err)
		}
	}

}

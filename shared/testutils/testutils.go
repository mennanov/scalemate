// Package testutils provides convenient functions for using in tests.
package testutils

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
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
	assert.Equal(t, code, statusCode.Code(), err)
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
func TruncateTables(db *sqlx.DB) {
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

// NewClientConn creates a new grpc.ClientConn to be used in tests.
func NewClientConn(addr string, creds credentials.TransportCredentials) *grpc.ClientConn {
	conn, err := grpc.DialContext(context.Background(), addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		panic(err)
	}

	return conn
}

// ExpectMessagesHandler is used in tests to check if all the expected messages have been received.
type ExpectMessagesHandler struct {
	expectedEventTypes []string
	done               chan struct{}
	doneClosed         bool
}

// NewExpectMessagesHandler creates a new instance of ExpectMessagesHandler.
func NewExpectMessagesHandler(expectedEventTypes ...string) *ExpectMessagesHandler {
	return &ExpectMessagesHandler{expectedEventTypes: expectedEventTypes, done: make(chan struct{})}
}

// ExpectMessages creates an event handler that expects the given messages and return a function that when is called
// waits for the remaining messages to be received or fails by the timeout.
// This method should be called BEFORE the events are expected to be produced.
func ExpectMessages(sc stan.Conn, subject string, logger *logrus.Logger, expectedEventTypes ...string) func(duration time.Duration) error {
	handler := NewExpectMessagesHandler(expectedEventTypes...)
	s, err := sc.Subscribe(subject, events.StanMsgHandler(context.Background(), logger, 0, handler), stan.SetManualAckMode())
	if err != nil {
		panic(err)
	}
	return func(d time.Duration) error {
		defer utils.Close(s, logger)
		return handler.Wait(d)
	}
}

// Handle deletes the received event types from the expectedEventTypes in the specified order.
func (h *ExpectMessagesHandler) Handle(_ context.Context, event *events_proto.Event) error {
	if len(h.expectedEventTypes) > 0 {
		actualEventType := fmt.Sprintf("%T", event.Payload)
		re := regexp.MustCompile(h.expectedEventTypes[0])
		if re.MatchString(actualEventType) {
			h.expectedEventTypes = h.expectedEventTypes[1:]
		}
	}
	if len(h.expectedEventTypes) == 0 {
		if !h.doneClosed {
			h.doneClosed = true
			close(h.done)
		}
	}
	return nil
}

// Wait waits for the expected messages to be received or returns an error if the timeout is hit.
func (h *ExpectMessagesHandler) Wait(timeout time.Duration) error {
	select {
	case <-time.After(timeout):
		return errors.Errorf("unmatched messages: %s", h.expectedEventTypes)

	case <-h.done:
		return nil
	}
}

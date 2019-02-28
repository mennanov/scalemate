package utils

import (
	"regexp"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
)

// WaitForMessages waits for the messages and matches their routing keys with the given keys.
// The function returns once all the provided keys have matched.
func WaitForMessages(messages <-chan amqp.Delivery, keys ...string) {
	allMessagesReceived := make(chan struct{})
	go func(c chan struct{}) {
		for msg := range messages {
			logrus.Debugf("Received AMQP message %s", msg.RoutingKey)
			for i := range keys {
				re := regexp.MustCompile(keys[i])
				if re.MatchString(msg.RoutingKey) {
					// Remove the key that matched from the keys.
					keys = append(keys[:i], keys[i+1:]...)
					if len(keys) == 0 {
						c <- struct{}{}
						return
					}
					break
				}
			}
		}
	}(allMessagesReceived)

	// Wait until all messages are received.
	<-allMessagesReceived
}

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


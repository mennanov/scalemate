package auth

import (
	"context"

	"github.com/dgrijalva/jwt-go"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Claims defines a JWT Scalemate.io specific claims.
type Claims struct {
	Username string
	// NodeName is used when authenticating Nodes. For clients it will be empty.
	NodeName string
	// A role defined in accounts.proto User message.
	Role      accounts_proto.User_Role
	TokenType string
	jwt.StandardClaims
}

// NewClaimsFromStringVerified parses the given token string and creates a new Claims struct.
// Token string is verified.
func NewClaimsFromStringVerified(tokenString string, jwtSecretKey []byte) (*Claims, error) {
	c := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, c, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "JWT can't be parsed: %s", err.Error())
	}

	if _, ok := token.Claims.(*Claims); !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid JWT claims")
	}
	return c, nil
}

// NewClaimsFromString parses a string token and created new claims, but DOES NOT verify the token.
// This method does not require a secret, thus should be used on clients to verify if the token has not expired
// before making an RPC.
func NewClaimsFromString(tokenString string) (*Claims, error) {
	c := &Claims{}
	parser := jwt.Parser{}
	_, _, err := parser.ParseUnverified(tokenString, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// SignedString creates a signed JWT string.
// Should be used on the server side to create new tokens.
func (c *Claims) SignedString(jwtSecretKey []byte) (string, error) {
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(jwtSecretKey)
	if err != nil {
		return "", status.Errorf(codes.Internal, "could not sign a JWT: %s", err.Error())
	}
	return tokenString, nil

}

type contextKey string

func (c contextKey) String() string {
	return "scalemate_" + string(c)
}

// ContextKeyClaims is a string key to be used to store and retrieve claims from the context.
var ContextKeyClaims = contextKey("claims")

// ParseJWTFromContext parses the JWT from the context. Returns a new context populated with claims from the JWT.
func ParseJWTFromContext(ctx context.Context, jwtSecretKey []byte) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	claims, err := NewClaimsFromStringVerified(token, jwtSecretKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	return context.WithValue(ctx, ContextKeyClaims, claims), nil
}

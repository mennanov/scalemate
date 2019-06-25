package auth

import (
	"context"

	"github.com/dgrijalva/jwt-go"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TokenType is a JWT type which can either be Access or Refresh (see below).
type TokenType int8

const (
	// TokenTypeAccess stands for an Access token type.
	TokenTypeAccess TokenType = 0
	// TokenTypeRefresh stands for a Refresh token type.
	TokenTypeRefresh TokenType = 1
)

// Claims defines a JWT Scalemate.io specific Claims.
type Claims struct {
	jwt.StandardClaims
	Username string
	// NodeName is used when authenticating Nodes. For clients it will be empty.
	NodeName  string
	TokenType TokenType
}

// NewClaims creates a new Claims instance.
func NewClaims(standardClaims jwt.StandardClaims, username string, nodeName string, tokenType TokenType) *Claims {
	return &Claims{StandardClaims: standardClaims, Username: username, NodeName: nodeName, TokenType: tokenType}
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
		return nil, status.Error(codes.InvalidArgument, "invalid JWT Claims")
	}
	return c, nil
}

// NewClaimsFromString parses a string token and created new Claims, but DOES NOT verify the token.
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

// ContextKeyClaims is a string key to be used to store and retrieve Claims from the context.
var ContextKeyClaims = contextKey("Claims")

// ClaimsInjector is an interface that is used to inject parsed and verified Claims to the context.
type ClaimsInjector interface {
	// Inject should extract Claims from the context, verify them and return a new context with the verified Claims set.
	InjectClaims(context.Context) (context.Context, error)
}

// JWTClaimsInjector implements ClaimsInjector interface for JWT.
type JWTClaimsInjector struct {
	jwtSecretKey []byte
}

// NewJWTClaimsInjector creates a new instance of JWTClaimsInjector for the provided secret key.
func NewJWTClaimsInjector(jwtSecretKey []byte) *JWTClaimsInjector {
	return &JWTClaimsInjector{jwtSecretKey: jwtSecretKey}
}

// InjectClaims parses the JWT from the context. Returns a new context populated with the verified Claims from the JWT.
func (i *JWTClaimsInjector) InjectClaims(ctx context.Context) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	claims, err := NewClaimsFromStringVerified(token, i.jwtSecretKey)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	return context.WithValue(ctx, ContextKeyClaims, claims), nil
}

// Compile time interface check.
var _ ClaimsInjector = new(JWTClaimsInjector)

// FakeClaimsInjector injects already provided Claims.
type FakeClaimsInjector struct {
	Claims   *Claims
}

// NewFakeClaimsContextInjector creates a new instance of NewFakeClaimsContextInjector.
func NewFakeClaimsContextInjector(claims *Claims) *FakeClaimsInjector {
	return &FakeClaimsInjector{Claims: claims}
}

// SetClaims sets the claims and returns a function that restores the original claims.
func (f *FakeClaimsInjector) SetClaims(claims *Claims) func() {
	originalClaims := f.Claims
	f.Claims = claims
	return func() {
		f.Claims = originalClaims
	}
}

// InjectClaims injects the provided Claims to the given context.
func (f *FakeClaimsInjector) InjectClaims(ctx context.Context) (context.Context, error) {
	return context.WithValue(ctx, ContextKeyClaims, f.Claims), nil
}

// Compile time interface check.
var _ ClaimsInjector = new(FakeClaimsInjector)

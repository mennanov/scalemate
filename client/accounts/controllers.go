package accounts

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/shared/auth"
)

var (
	// ErrEmptyUsername is used when a username is expected, but it is empty.
	ErrEmptyUsername = errors.New("username can not be empty")
	// ErrEmptyPassword is used when a password is expected, but it is empty.
	ErrEmptyPassword = errors.New("password can not be empty")
	// ErrEmptyEmail is used when an email is expected, but it is empty.
	ErrEmptyEmail = errors.New("email can not be empty")
)

// LoginController performs a gRPC call to verify user credentials. If everything is OK it saves AuthTokens
// so that they can be reused later.
func LoginController(
	client accounts_proto.AccountsClient,
	username,
	password string,
) (*accounts_proto.AuthTokens, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}
	request := &accounts_proto.PasswordAuthRequest{
		Username: username,
		Password: password,
	}
	if err := request.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid arguments format")
	}
	tokens, err := client.PasswordAuth(context.Background(), request)
	if err != nil {
		return nil, err
	}
	// Save tokens to a tmp file.
	if err := auth.SaveTokens(tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// LogoutController deletes the file that stores AuthTokens making it impossible to make RPCs afterwards.
func LogoutController() error {
	return errors.Wrap(auth.DeleteTokens(), "could not delete saved auth tokens. Are you logged in?")
}

// RegisterController registers a new user account. User needs to login afterwards.
func RegisterController(client accounts_proto.AccountsClient, username, email, password string) error {
	if username == "" {
		return ErrEmptyUsername
	}
	if email == "" {
		return ErrEmptyEmail
	}
	if password == "" {
		return ErrEmptyPassword
	}

	request := &accounts_proto.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}
	if err := request.Validate(); err != nil {
		return errors.Wrap(err, "invalid arguments format")
	}
	_, err := client.Register(context.Background(), request)
	return err
}

// ChangePasswordController changes a password for the currently logged in user.
func ChangePasswordController(client accounts_proto.AccountsClient, password string) error {
	if password == "" {
		return ErrEmptyPassword
	}

	tokens, err := auth.LoadTokens()
	if err != nil {
		return err
	}
	jwtCredentials := auth.NewJWTCredentials(client, tokens, auth.SaveTokens)
	claims, err := auth.NewClaimsFromString(tokens.AccessToken)
	if err != nil {
		return err
	}

	request := &accounts_proto.ChangePasswordRequest{
		Username: claims.Username,
		Password: password,
	}

	_, err = client.ChangePassword(context.Background(), request, grpc.PerRPCCredentials(jwtCredentials))
	return err
}

package accounts_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/client/accounts"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

func TestLoginController(t *testing.T) {
	t.Run("EmptyUsername", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		p, err := accounts.LoginController(NewMockAccountsClient(ctrl), "", "password")
		require.EqualError(t, err, accounts.ErrEmptyUsername.Error())
		assert.Nil(t, p)
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		p, err := accounts.LoginController(NewMockAccountsClient(ctrl), "username", "")
		require.EqualError(t, err, accounts.ErrEmptyPassword.Error())
		assert.Nil(t, p)
	})

	t.Run("InvalidInput", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		p, err := accounts.LoginController(
			NewMockAccountsClient(ctrl), "invalid username", "some password")
		require.Error(t, err)
		assert.Nil(t, p)
	})

	t.Run("LoginOK", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		username := "username"
		password := "password"
		ctx := context.Background()
		authTokens := &accounts_proto.AuthTokens{AccessToken: "accessToken", RefreshToken: "refreshToken"}
		client := NewMockAccountsClient(ctrl)
		client.EXPECT().
			PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{Username: username, Password: password}).
			Return(authTokens, nil)

		r, err := accounts.LoginController(client, username, password)
		require.NoError(t, err)
		assert.Equal(t, authTokens.AccessToken, r.AccessToken)
		assert.Equal(t, authTokens.RefreshToken, r.RefreshToken)
		// Verify that the tokens have been saved.
		loadedTokens, err := auth.LoadTokens()
		require.NoError(t, err)
		assert.Equal(t, authTokens.AccessToken, loadedTokens.AccessToken)
		assert.Equal(t, authTokens.RefreshToken, loadedTokens.RefreshToken)
	})

	t.Run("LoginFail", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		username := "username"
		password := "invalid password"
		ctx := context.Background()

		client := NewMockAccountsClient(ctrl)
		client.EXPECT().
			PasswordAuth(ctx, &accounts_proto.PasswordAuthRequest{Username: username, Password: password}).
			Return(nil, status.Error(codes.InvalidArgument, "invalid password"))

		r, err := accounts.LoginController(client, username, password)
		require.Error(t, err)
		assert.Nil(t, r)
	})
}

func TestLogoutController(t *testing.T) {
	t.Run("LogoutOk", func(t *testing.T) {
		authTokens := &accounts_proto.AuthTokens{AccessToken: "accessToken", RefreshToken: "refreshToken"}
		err := auth.SaveTokens(authTokens)
		require.NoError(t, err)
		err = accounts.LogoutController()
		require.NoError(t, err)
		r, err := auth.LoadTokens()
		require.Error(t, err)
		assert.Nil(t, r)
	})

	t.Run("LogoutFail", func(t *testing.T) {
		err := accounts.LogoutController()
		require.Error(t, err)
	})
}

func TestRegisterController(t *testing.T) {
	t.Run("EmptyUsername", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		err := accounts.RegisterController(NewMockAccountsClient(ctrl), "", "email@email.com", "password")
		require.EqualError(t, err, accounts.ErrEmptyUsername.Error())
	})

	t.Run("EmptyEmail", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		err := accounts.RegisterController(NewMockAccountsClient(ctrl), "username", "", "password")
		require.EqualError(t, err, accounts.ErrEmptyEmail.Error())
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		err := accounts.RegisterController(NewMockAccountsClient(ctrl), "username", "email@email.com", "")
		require.EqualError(t, err, accounts.ErrEmptyPassword.Error())
	})

	t.Run("InvalidInput", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		err := accounts.RegisterController(
			NewMockAccountsClient(ctrl), "invalid username", "invalid email", "some password")
		require.Error(t, err)
	})

	t.Run("RegisterOK", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		username := "username"
		email := "email@email.com"
		password := "password"
		ctx := context.Background()
		client := NewMockAccountsClient(ctrl)
		client.EXPECT().
			Register(ctx, &accounts_proto.RegisterRequest{Username: username, Email: email, Password: password}).
			Return(&empty.Empty{}, nil)

		err := accounts.RegisterController(client, username, email, password)
		assert.NoError(t, err)
	})

	t.Run("RegisterFail", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		username := "existing_username"
		email := "email@email.com"
		password := "password"
		ctx := context.Background()

		client := NewMockAccountsClient(ctrl)
		client.EXPECT().
			Register(ctx, &accounts_proto.RegisterRequest{Username: username, Email: email, Password: password}).
			Return(nil, status.Error(codes.AlreadyExists, "already registered"))

		err := accounts.RegisterController(client, username, email, password)
		assert.Error(t, err)
	})
}

func TestChangePasswordController(t *testing.T) {
	t.Run("EmptyPassword", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		err := accounts.ChangePasswordController(NewMockAccountsClient(ctrl), "")
		require.EqualError(t, err, accounts.ErrEmptyPassword.Error())
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := NewMockAccountsClient(ctrl)

		err := accounts.ChangePasswordController(client, "password")
		assert.Error(t, err)
	})

	t.Run("InvalidAuthTokens", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authTokens := &accounts_proto.AuthTokens{
			AccessToken:  "invalid jwt access token",
			RefreshToken: "invalid jwt refresh token",
		}
		err := auth.SaveTokens(authTokens)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, auth.DeleteTokens())
		}()

		client := NewMockAccountsClient(ctrl)

		err = accounts.ChangePasswordController(client, "password")
		assert.Error(t, err)
	})

	t.Run("ChangePasswordOK", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "username")
		defer deleteTokens()

		ctx := context.Background()
		client := NewMockAccountsClient(ctrl)
		client.EXPECT().
			ChangePassword(ctx, &accounts_proto.ChangePasswordRequest{Username: "username", Password: "password"}, gomock.Any()).
			Return(&empty.Empty{}, nil)

		err := accounts.ChangePasswordController(client, "password")
		assert.NoError(t, err)
	})

	t.Run("ChangePasswordFail", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "username")
		defer deleteTokens()

		ctx := context.Background()
		client := NewMockAccountsClient(ctrl)
		client.EXPECT().
			ChangePassword(ctx, &accounts_proto.ChangePasswordRequest{Username: "username", Password: "password"}, gomock.Any()).
			Return(&empty.Empty{}, status.Error(codes.PermissionDenied, "permissions denied"))

		err := accounts.ChangePasswordController(client, "password")
		assert.Error(t, err)
	})
}

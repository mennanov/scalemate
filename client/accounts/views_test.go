package accounts_test

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/mennanov/scalemate/client/accounts"
	"github.com/mennanov/scalemate/shared/testutils"
)

func genericViewsTest(t *testing.T, view func(logger *logrus.Logger, err error)) {
	t.Run("ErrorIsLogged", func(t *testing.T) {
		for _, err := range testutils.GetAllErrors() {
			logger, hook := test.NewNullLogger()
			view(logger, err)
			assert.Equal(t, 1, len(hook.Entries))
			assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level, "failed for err %s", err)
		}
	})

	t.Run("SuccessMessageIsLogged", func(t *testing.T) {
		logger, hook := test.NewNullLogger()
		view(logger, nil)
		assert.Equal(t, 1, len(hook.Entries))
		assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	})
}

func TestLoginView(t *testing.T) {
	genericViewsTest(t, accounts.LoginView)
}

func TestChangePasswordView(t *testing.T) {
	genericViewsTest(t, accounts.ChangePasswordView)
}

func TestLogoutView(t *testing.T) {
	genericViewsTest(t, accounts.LogoutView)
}

func TestRegisterView(t *testing.T) {
	genericViewsTest(t, accounts.RegisterView)
}

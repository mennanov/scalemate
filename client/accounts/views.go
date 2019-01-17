package accounts

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/client"
)

// LoginView handles LoginController results representation.
func LoginView(logger *logrus.Logger, err error) {
	if err != nil {
		client.ErrorView(logger, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid login credentials: %s", s.Message())
			},
			NotFound: func(s *status.Status) string {
				return "user with the given username was not found"
			},
			PermissionDenied: func(s *status.Status) string {
				return fmt.Sprintf("permission denied: %s", s.Message())
			},
		}, err)
		return
	}
	logger.Info("Logged in.")
}

// RegisterView handles RegisterController results representation.
func RegisterView(logger *logrus.Logger, err error) {
	if err != nil {
		client.ErrorView(logger, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid parameters: %s", s.Message())
			},
			AlreadyExists: func(s *status.Status) string {
				return "user already exists. Please, login."
			},
		}, err)
		return
	}
	logger.Info("Registration complete. You may now login.")
}

// ChangePasswordView handles ChangePasswordController results representation.
func ChangePasswordView(logger *logrus.Logger, err error) {
	if err != nil {
		client.ErrorView(logger, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid parameters: %s", s.Message())
			},
			PermissionDenied: func(s *status.Status) string {
				return fmt.Sprintf("permission denied: %s", s.Message())
			},
		}, err)
		return
	}
	logger.Info("Password has been changed. Please, log in.")
}

// LogoutView handles LogoutController representation.
func LogoutView(logger *logrus.Logger, err error) {
	if err != nil {
		client.ErrorView(logger, &client.GRPCErrorMessages{}, err)
		return
	}
	logger.Info("Logged out.")
}

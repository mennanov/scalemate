package accounts

import (
	"fmt"
	"io"

	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/client"
)

// LoginView handles LoginController results representation.
func LoginView(outWriter, errWriter io.Writer, err error) {
	if err != nil {
		client.ErrorView(errWriter, &client.GRPCErrorMessages{
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
	client.DisplayMessage(outWriter, "Logged in.")
}

// RegisterView handles RegisterController results representation.
func RegisterView(outWriter, errWriter io.Writer, err error) {
	if err != nil {
		client.ErrorView(errWriter, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid parameters: %s", s.Message())
			},
			AlreadyExists: func(s *status.Status) string {
				return "user already exists. Please, login."
			},
		}, err)
		return
	}
	client.DisplayMessage(outWriter, "Registration complete. You may now login.")
}

// ChangePasswordView handles ChangePasswordController results representation.
func ChangePasswordView(outWriter, errWriter io.Writer, err error) {
	if err != nil {
		client.ErrorView(errWriter, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid parameters: %s", s.Message())
			},
			PermissionDenied: func(s *status.Status) string {
				return fmt.Sprintf("permission denied: %s", s.Message())
			},
		}, err)
		return
	}
	client.DisplayMessage(outWriter, "Password has been changed. Please, log in.")
}

// LogoutView handles LogoutController representation.
func LogoutView(outWriter, errWriter io.Writer, err error) {
	if err != nil {
		client.ErrorView(errWriter, &client.GRPCErrorMessages{}, err)
		return
	}
	client.DisplayMessage(outWriter, "Logged out.")
}

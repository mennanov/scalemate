package server

import (
	"context"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	// required by gorm
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/middleware"
)

// LoggedErrorCodes are the error codes for the errors that will be logged with the "Error" level with a full stack
// trace if available.
var LoggedErrorCodes = []codes.Code{
	codes.Unknown,
	codes.Internal,
	codes.DeadlineExceeded,
	codes.DataLoss,
	codes.FailedPrecondition,
	codes.Aborted,
	codes.OutOfRange,
	codes.ResourceExhausted,
	codes.Unavailable,
	codes.Unimplemented,
	codes.PermissionDenied,
}

// AccountsServer implements Accounts gRPC service.
type AccountsServer struct {
	db             *sqlx.DB
	claimsInjector auth.ClaimsInjector
	producer       events.Producer
	// bcrypt cost value used to make password hashes. Should be reasonably high in PROD and low in TEST/DEV.
	bCryptCost      int
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	jwtSecretKey    []byte
	logger          *logrus.Logger
}

// Compile time interface check.
var _ accounts_proto.AccountsServer = new(AccountsServer)

// NewAccountsServer creates a new AccountsServer and applies the given options to it.
func NewAccountsServer(options ...Option) (*AccountsServer, error) {
	s := &AccountsServer{}

	for _, option := range options {
		if err := option(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Serve creates a GRPC server and starts serving on a given address. This function is blocking and runs upon the server
// termination (when the context is Done).
func (s *AccountsServer) Serve(ctx context.Context, grpcAddr string, creds credentials.TransportCredentials) {
	entry := logrus.NewEntry(s.logger)
	grpc_logrus.ReplaceGrpcLogger(entry)
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(entry),
			middleware.LoggerRequestIDInterceptor("request.id"),
			grpc_auth.UnaryServerInterceptor(AuthFunc),
			grpc_validator.UnaryServerInterceptor(),
			middleware.StackTraceErrorInterceptor(false, LoggedErrorCodes...),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	accounts_proto.RegisterAccountsServer(grpcServer, s)

	serverErrors := make(chan error, 1)

	go func(f chan error) {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			panic(err)
		}
		s.logger.Infof("Serving on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			f <- err
		}
	}(serverErrors)

	select {
	case <-ctx.Done():
		s.logger.Info("Gracefully stopping gRPC server...")
		grpcServer.GracefulStop()

	case err := <-serverErrors:
		s.logger.WithError(err).Error("gRPC server unexpectedly failed.")
	}
	s.logger.Info("gRPC server is stopped.")
}

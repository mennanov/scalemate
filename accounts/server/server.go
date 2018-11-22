package server

import (
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	// required by gorm
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/middleware"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
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

// AMQPEnvConf maps to the name of the env variable with the AMQP address to connect to.
var AMQPEnvConf = utils.AMQPEnvConf{
	Addr: "SHARED_AMQP_ADDR",
}

// TLSEnvConf maps TLS env variables.
var TLSEnvConf = utils.TLSEnvConf{
	CertFile: "ACCOUNTS_TLS_CERT_FILE",
	KeyFile:  "ACCOUNTS_TLS_KEY_FILE",
}

// DBEnvConf maps to the name of the env variable with the Postgres address to connect to.
var DBEnvConf = utils.DBEnvConf{
	Host:     "ACCOUNTS_DB_HOST",
	Port:     "ACCOUNTS_DB_PORT",
	User:     "ACCOUNTS_DB_USER",
	Name:     "ACCOUNTS_DB_NAME",
	Password: "ACCOUNTS_DB_PASSWORD",
}

// AppEnvConf represents other application settings env variable mapping.
type AppEnvConf struct {
	BCryptCost      string
	AccessTokenTTL  string
	RefreshTokenTTL string
	JWTSecretKey    string
	TLSCert         string
	TLSKey          string
}

// AccountsEnvConf maps env variables for the Accounts service.
var AccountsEnvConf = AppEnvConf{
	BCryptCost:      "ACCOUNTS_BCRYPT_COST",
	AccessTokenTTL:  "ACCOUNTS_ACCESS_TOKEN_TTL",
	RefreshTokenTTL: "ACCOUNTS_REFRESH_TOKEN_TTL",
	JWTSecretKey:    "ACCOUNTS_JWT_SECRET_KEY",
}

// AccountsServer is a wrapper for `accounts_proto.AccountsServer` that holds the application specific settings
// and also implements some methods.
type AccountsServer struct {
	DB             *gorm.DB
	AMQPConnection *amqp.Connection
	// bcrypt cost value used to make password hashes. Should be reasonably high in PROD and low in TEST/DEV.
	BcryptCost      int
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	JWTSecretKey    []byte
}

// Compile time interface check.
var _ accounts_proto.AccountsServer = new(AccountsServer)

// NewAccountServerFromEnv create a new AccountsServer from environment variables.
// It panics if some env vars are missing or have invalid values.
func NewAccountServerFromEnv(conf AppEnvConf) (*AccountsServer, error) {

	bcryptCost, err := strconv.Atoi(os.Getenv(conf.BCryptCost))

	if err != nil {
		return nil, errors.Wrapf(err, "invalid value '%s' for %s", os.Getenv(conf.BCryptCost), conf.BCryptCost)
	}

	accessTokenTTL, err := time.ParseDuration(os.Getenv(conf.AccessTokenTTL))

	if err != nil {
		return nil, errors.Wrapf(err, "invalid value '%s' for %s", os.Getenv(conf.AccessTokenTTL),
			conf.AccessTokenTTL)
	}

	refreshTokenTTL, err := time.ParseDuration(os.Getenv(conf.RefreshTokenTTL))

	if err != nil {
		return nil, errors.Wrapf(err, "invalid value '%s' for %s", os.Getenv(conf.RefreshTokenTTL),
			conf.RefreshTokenTTL)
	}

	jwtSecretKey := os.Getenv(conf.JWTSecretKey)

	if jwtSecretKey == "" {
		return nil, errors.Errorf("%s env variable is empty", conf.JWTSecretKey)
	}

	db, err := utils.ConnectDBFromEnv(DBEnvConf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	amqpConnection, err := utils.ConnectAMQPFromEnv(AMQPEnvConf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to AMQP")
	}

	channel, err := amqpConnection.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AMQP channel")
	}
	// This channel is only used to declare the exchange.
	defer utils.Close(channel)

	if err := utils.AMQPExchangeDeclare(channel, utils.AccountsAMQPExchangeName); err != nil {
		return nil, errors.Wrap(err, "failed to declare AMQP exchange")
	}

	return &AccountsServer{
		BcryptCost:      bcryptCost,
		AccessTokenTTL:  accessTokenTTL,
		RefreshTokenTTL: refreshTokenTTL,
		JWTSecretKey:    []byte(jwtSecretKey),
		DB:              db,
		AMQPConnection:  amqpConnection,
	}, nil
}

// Close closes the server resources.
func (s *AccountsServer) Close() error {
	return s.DB.Close()
}

// Serve creates a GRPC server and starts serving on a given address. This function is blocking and runs upon the server
// termination.
func Serve(grpcAddr string, srv *AccountsServer) {
	setLogLevelFromEnv()

	logrusEntry := logrus.NewEntry(logrus.StandardLogger())
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	opts := []grpc_ctxtags.Option{
		grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.TagBasedRequestFieldExtractor("log_fields")),
	}
	grpcServer := grpc.NewServer(
		grpc.Creds(utils.TLSServerCredentialsFromEnv(TLSEnvConf)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(opts...),
			grpc_logrus.UnaryServerInterceptor(logrusEntry),
			// grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry),
			grpc_auth.UnaryServerInterceptor(AuthFunc),
			grpc_validator.UnaryServerInterceptor(),
			middleware.StackTraceErrorInterceptor(false, LoggedErrorCodes...),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	accounts_proto.RegisterAccountsServer(grpcServer, srv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	f := make(chan error, 1)

	go func(f chan error) {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			panic(err)
		}
		logrus.Infof("Serving on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			f <- err
		}
	}(f)

	// Handle events.
	go func(f chan error) {
		if err := srv.HandleNodeCreatedEvents(); err != nil {
			f <- err
		}
	}(f)

	select {
	case <-c:
		grpcServer.GracefulStop()

	case err := <-f:
		logrus.Fatalf("GRPC server unexpectedly failed: %s", err)
	}
}

func setLogLevelFromEnv() {
	lvl := os.Getenv("LOG_LEVEL")
	switch lvl {
	case "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}

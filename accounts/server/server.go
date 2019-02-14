package server

import (
	"context"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"google.golang.org/grpc/codes"
	// required by gorm
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/middleware"
	"github.com/mennanov/scalemate/shared/utils"
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
	ClaimsInjector auth.ClaimsInjector
	Producer       events.Producer
	Consumers      []events.Consumer
	// bcrypt cost value used to make password hashes. Should be reasonably high in PROD and low in TEST/DEV.
	BcryptCost      int
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	JWTSecretKey    []byte
	logger          *logrus.Logger
}

// WithLogger creates an option that sets the logger.
func WithLogger(logger *logrus.Logger) Option {
	return func(s *AccountsServer) error {
		s.logger = logger
		return nil
	}
}

// WithBCryptCostFromEnv sets the BcryptCost field value from environment variables.
func WithBCryptCostFromEnv(conf AppEnvConf) Option {
	return func(s *AccountsServer) error {
		value, err := strconv.Atoi(os.Getenv(conf.BCryptCost))
		if err != nil {
			return errors.Wrap(err, "strconv.Atoi failed")
		}
		s.BcryptCost = value
		return nil
	}
}

// WithAccessTokenTTLFromEnv sets the AccessTokenTTL field value from environment variables.
func WithAccessTokenTTLFromEnv(conf AppEnvConf) Option {
	return func(s *AccountsServer) error {
		value, err := time.ParseDuration(os.Getenv(conf.AccessTokenTTL))
		if err != nil {
			return errors.Wrapf(err, "time.ParseDuration failed for '%s'", os.Getenv(conf.AccessTokenTTL))
		}
		s.AccessTokenTTL = value
		return nil
	}
}

// WithRefreshTokenTTLFromEnv creates an option that sets the RefreshTokenTTL field value from environment variables.
func WithRefreshTokenTTLFromEnv(conf AppEnvConf) Option {
	return func(s *AccountsServer) error {
		value, err := time.ParseDuration(os.Getenv(conf.RefreshTokenTTL))
		if err != nil {
			return errors.Wrapf(err, "time.ParseDuration failed for '%s'", os.Getenv(conf.RefreshTokenTTL))
		}
		s.RefreshTokenTTL = value
		return nil
	}
}

// JWTSecretKeyFromEnv gets the JWT secret key value from environment variables.
func JWTSecretKeyFromEnv(conf AppEnvConf) ([]byte, error) {
	value := os.Getenv(conf.JWTSecretKey)
	if value == "" {
		return nil, errors.New("JWT secret key is empty")
	}
	return []byte(value), nil
}

// WithJWTSecretKey creates an option that sets the JWTSecretKey field value.
func WithJWTSecretKey(jwtSecretKey []byte) Option {
	return func(s *AccountsServer) error {
		s.JWTSecretKey = jwtSecretKey
		return nil
	}
}

// WithClaimsInjector creates an option that sets the ClaimsInjector to auth.NewJWTClaimsInjector with the
// provided jwtSecretKey.
func WithClaimsInjector(jwtSecretKey []byte) Option {
	return func(s *AccountsServer) error {
		s.ClaimsInjector = auth.NewJWTClaimsInjector(jwtSecretKey)
		return nil
	}
}

// WithAMQPProducer creates an option that sets the Producer field value to AMQPProducer.
func WithAMQPProducer(conn *amqp.Connection) Option {
	return func(s *AccountsServer) error {
		if conn == nil {
			return errors.New("amqp.Connection is nil")
		}
		producer, err := events.NewAMQPProducer(conn, events.AccountsAMQPExchangeName)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPProducer failed")
		}
		s.Producer = producer
		return nil
	}
}

// WithAMQPConsumers creates an option that sets the Consumers field value to AMQPConsumer(s).
func WithAMQPConsumers(conn *amqp.Connection) Option {
	return func(s *AccountsServer) error {
		channel, err := conn.Channel()
		defer utils.Close(channel)

		if err != nil {
			return errors.Wrap(err, "failed to open a new AMQP channel")
		}
		// Declare all required exchanges.
		if err := events.AMQPExchangeDeclare(channel, events.AccountsAMQPExchangeName); err != nil {
			return errors.Wrapf(err, "failed to declare AMQP exchange %s", events.AccountsAMQPExchangeName)
		}
		if err := events.AMQPExchangeDeclare(channel, events.SchedulerAMQPExchangeName); err != nil {
			return errors.Wrapf(err, "failed to declare AMQP exchange %s", events.SchedulerAMQPExchangeName)
		}

		nodeConnectedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"accounts_scheduler_node_created",
			"scheduler.node.created",
			s.HandleSchedulerNodeCreatedEvent)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for nodeConnectedConsumer")
		}
		s.Consumers = []events.Consumer{nodeConnectedConsumer}

		return nil
	}
}

// WithDBConnection creates an option that sets the DB field to an existing DB connection.
func WithDBConnection(db *gorm.DB) Option {
	return func(s *AccountsServer) error {
		s.DB = db
		return nil
	}
}

// Compile time interface check.
var _ accounts_proto.AccountsServer = new(AccountsServer)

// Option modifies the SchedulerServer.
type Option func(server *AccountsServer) error

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

// Close closes the server resources.
func (s *AccountsServer) Close() error {
	for _, consumer := range s.Consumers {
		if err := consumer.Close(); err != nil {
			return errors.Wrap(err, "consumer.Close failed")
		}
	}

	return errors.Wrap(s.Producer.Close(), "AccountsServer.Producer.Close failed")
}

// Serve creates a GRPC server and starts serving on a given address. This function is blocking and runs upon the server
// termination.
func (s *AccountsServer) Serve(grpcAddr string, shutdown chan os.Signal) {
	entry := logrus.NewEntry(s.logger)
	grpc_logrus.ReplaceGrpcLogger(entry)
	grpcServer := grpc.NewServer(
		grpc.Creds(utils.TLSServerCredentialsFromEnv(TLSEnvConf)),
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

	f := make(chan error, 1)

	go func(f chan error) {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			panic(err)
		}
		s.logger.Infof("Serving on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			f <- err
		}
	}(f)

	wg := &sync.WaitGroup{}
	consumersCtx, consumersCtxCancel := context.WithCancel(context.Background())
	for _, consumer := range s.Consumers {
		go consumer.Consume(consumersCtx, wg)
	}

	defer func() {
		consumersCtxCancel()
		wg.Wait()
	}()

	select {
	case <-shutdown:
		s.logger.Info("Gracefully stopping gRPC server...")
		grpcServer.GracefulStop()
		s.logger.Info("gRPC stopped.")

	case err := <-f:
		s.logger.WithError(err).Error("gRPC server unexpectedly failed")
	}
}

package server

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
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
	DB                    *gorm.DB
	ClaimsContextInjector auth.ClaimsInjector
	Producer              events.Producer
	Consumers             []events.Consumer
	// bcrypt cost value used to make password hashes. Should be reasonably high in PROD and low in TEST/DEV.
	BcryptCost      int
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	JWTSecretKey    []byte

	amqpConnection *amqp.Connection
}

// SetBCryptCostFromEnv sets the BcryptCost field value from environment variables.
func (s *AccountsServer) SetBCryptCostFromEnv(conf AppEnvConf) error {
	value, err := strconv.Atoi(os.Getenv(conf.BCryptCost))
	if err != nil {
		return errors.Wrap(err, "strconv.Atoi failed")
	}
	s.BcryptCost = value
	return nil
}

// SetAccessTokenTTLFromEnv sets the AccessTokenTTL field value from environment variables.
func (s *AccountsServer) SetAccessTokenTTLFromEnv(conf AppEnvConf) error {
	value, err := time.ParseDuration(os.Getenv(conf.AccessTokenTTL))
	if err != nil {
		return errors.Wrapf(err, "time.ParseDuration failed for '%s'", os.Getenv(conf.AccessTokenTTL))
	}
	s.AccessTokenTTL = value
	return nil
}

// SetRefreshTokenTTLFromEnv sets the RefreshTokenTTL field value from environment variables.
func (s *AccountsServer) SetRefreshTokenTTLFromEnv(conf AppEnvConf) error {
	value, err := time.ParseDuration(os.Getenv(conf.RefreshTokenTTL))
	if err != nil {
		return errors.Wrapf(err, "time.ParseDuration failed for '%s'", os.Getenv(conf.RefreshTokenTTL))
	}
	s.RefreshTokenTTL = value
	return nil
}

// SetJWTSecretKeyFromEnv sets the JWTSecretKey field value from environment variables.
func (s *AccountsServer) SetJWTSecretKeyFromEnv(conf AppEnvConf) error {
	value := os.Getenv(conf.JWTSecretKey)
	if value == "" {
		return errors.New("JWT secret key is empty")
	}
	s.JWTSecretKey = []byte(value)
	return nil
}

// SetClaimsContextInjector sets the ClaimsContextInjector to auth.NewJWTClaimsInjector with provided jwtSecretKey.
func (s *AccountsServer) SetClaimsContextInjector(jwtSecretKey []byte) error {
	s.ClaimsContextInjector = auth.NewJWTClaimsInjector(jwtSecretKey)
	return nil
}

// SetDBConnectionFromEnv connects to DB with configuration from environment variables and sets the DB field value.
func (s *AccountsServer) SetDBConnectionFromEnv(conf utils.DBEnvConf) error {
	db, err := utils.ConnectDBFromEnv(conf)
	if err != nil {
		return errors.Wrap(err, "utils.ConnectDBFromEnv failed")
	}
	s.DB = db
	return nil
}

// SetAMQPConnectionFromEnv connects to AMQP with configuration from environment variables and sets the field value.
func (s *AccountsServer) SetAMQPConnectionFromEnv(amqpEnvConf utils.AMQPEnvConf) error {
	connection, err := utils.ConnectAMQPFromEnv(amqpEnvConf)
	if err != nil {
		return errors.Wrap(err, "utils.ConnectAMQPFromEnv failed")
	}
	s.amqpConnection = connection
	return nil
}

// SetAMQPProducer sets the Producer field value to AMQPProducer.
func (s *AccountsServer) SetAMQPProducer(conn *amqp.Connection) error {
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

// SetAMQPConsumers sets the Consumers field value to AMQPConsumer(s).
func (s *AccountsServer) SetAMQPConsumers(conn *amqp.Connection) error {
	channel, err := s.amqpConnection.Channel()
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
		s.amqpConnection,
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

// RunConsumers starts consuming events for all consumers.
func (s *AccountsServer) RunConsumers(ctx context.Context, wg *sync.WaitGroup) {
	for _, consumer := range s.Consumers {
		go consumer.Consume(ctx, wg)
	}
}

// Compile time interface check.
var _ accounts_proto.AccountsServer = new(AccountsServer)

// NewAccountServerFromEnv create a new AccountsServer suitable for production from environment variables.
func NewAccountServerFromEnv(appEnvConf AppEnvConf, dbEnvConf utils.DBEnvConf, amqpEnvConf utils.AMQPEnvConf) (*AccountsServer, error) {
	s := &AccountsServer{}
	if err := s.SetAccessTokenTTLFromEnv(appEnvConf); err != nil {
		return nil, errors.Wrap(err, "SetAccessTokenTTLFromEnv failed")
	}
	if err := s.SetRefreshTokenTTLFromEnv(appEnvConf); err != nil {
		return nil, errors.Wrap(err, "SetRefreshTokenTTLFromEnv failed")
	}
	if err := s.SetBCryptCostFromEnv(appEnvConf); err != nil {
		return nil, errors.Wrap(err, "SetBCryptCostFromEnv failed")
	}
	if err := s.SetJWTSecretKeyFromEnv(appEnvConf); err != nil {
		return nil, errors.Wrap(err, "SetJWTSecretKeyFromEnv failed")
	}
	if err := s.SetClaimsContextInjector(s.JWTSecretKey); err != nil {
		return nil, errors.Wrap(err, "SetClaimsContextInjector failed")
	}
	if err := s.SetDBConnectionFromEnv(dbEnvConf); err != nil {
		return nil, errors.Wrap(err, "SetDBConnectionFromEnv failed")
	}
	if err := s.SetAMQPConnectionFromEnv(amqpEnvConf); err != nil {
		return nil, errors.Wrap(err, "SetAMQPConnectionFromEnv failed")
	}
	if err := s.SetAMQPProducer(s.amqpConnection); err != nil {
		return nil, errors.Wrap(err, "SetAMQPProducer failed")
	}
	if err := s.SetAMQPConsumers(s.amqpConnection); err != nil {
		return nil, errors.Wrap(err, "SetAMQPConsumers failed")
	}
	return s, nil
}

// Close closes the server resources.
func (s *AccountsServer) Close() error {
	if err := s.DB.Close(); err != nil {
		return errors.Wrap(err, "AccountsServer.DB.Close failed")
	}
	if err := s.Producer.Close(); err != nil {
		return errors.Wrap(err, "AccountsServer.Producer.Close failed")
	}
	for _, consumer := range s.Consumers {
		if err := consumer.Close(); err != nil {
			return errors.Wrap(err, "consumer.Close failed")
		}
	}

	return errors.Wrap(s.amqpConnection.Close(), "AccountsServer.amqpConnection.Close failed")
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

	wg := &sync.WaitGroup{}
	consumersCtx, consumersCtxCancel := context.WithCancel(context.Background())
	srv.RunConsumers(consumersCtx, wg)

	select {
	case <-c:
		grpcServer.GracefulStop()
		consumersCtxCancel()
		wg.Wait()

	case err := <-f:
		consumersCtxCancel()
		wg.Wait()
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

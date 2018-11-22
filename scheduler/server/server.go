package server

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/shared/utils"
	"google.golang.org/grpc/codes"

	// required by gorm
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/middleware"
	"github.com/pkg/errors"
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

// SchedulerServer is a wrapper for `scheduler_proto.SchedulerServer` that holds the application specific settings
// and also implements some methods.
type SchedulerServer struct {
	DB             *gorm.DB
	JWTSecretKey   []byte
	AMQPConnection *amqp.Connection
	// Nodes that are currently connected to this service instance. By sending a Task message to this channel the Task
	// will be started on the Node with Node.ID as the map key.
	ConnectedNodes map[uint64]chan *scheduler_proto.Task
	// Jobs that are not scheduled yet and are awaiting to be scheduled. By sending a Task message to this channel the
	// client (which has created a corresponding Job with Job.ID as the map key) will be notified that the Task is
	// running on some Node.
	AwaitingJobs map[uint64]chan *scheduler_proto.Task
	// gracefulStop channel is used to notify about the gRPC server GracefulStop() in progress. When the server is about
	// to stop this channel is closed, this will unblock all the receivers of this channel and they will receive a zero
	// value (empty struct) which should be discarded and take an appropriate action to gracefully finish long-running
	// request processing (streams, etc).
	// Requests that are expected to be processed quickly should not take any action as they will be waited to finish.
	gracefulStop chan struct{}
}

// Compile time interface check.
var _ scheduler_proto.SchedulerServer = new(SchedulerServer)

// AMQPEnvConf maps to the name of the env variable with the AMQP address to connect to.
var AMQPEnvConf = utils.AMQPEnvConf{
	Addr: "SHARED_AMQP_ADDR",
}

// TLSEnvConf maps TLS env variables.
var TLSEnvConf = utils.TLSEnvConf{
	CertFile: "SCHEDULER_TLS_CERT_FILE",
	KeyFile:  "SCHEDULER_TLS_KEY_FILE",
}

// DBEnvConf maps to the name of the env variable with the Postgres address to connect to.
var DBEnvConf = utils.DBEnvConf{
	Host:     "SCHEDULER_DB_HOST",
	Port:     "SCHEDULER_DB_PORT",
	User:     "SCHEDULER_DB_USER",
	Name:     "SCHEDULER_DB_NAME",
	Password: "SCHEDULER_DB_PASSWORD",
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

// SchedulerEnvConf maps env variables for the Accounts service.
var SchedulerEnvConf = AppEnvConf{
	BCryptCost:      "SCHEDULER_BCRYPT_COST",
	AccessTokenTTL:  "SCHEDULER_ACCESS_TOKEN_TTL",
	RefreshTokenTTL: "SCHEDULER_REFRESH_TOKEN_TTL",
	JWTSecretKey:    "SCHEDULER_JWT_SECRET_KEY",
}

// NewSchedulerServerFromEnv creates a new SchedulerServer from environment variables.
// It also connects to the database and AMQP.
func NewSchedulerServerFromEnv(conf AppEnvConf) (*SchedulerServer, error) {
	jwtSecretKey := os.Getenv(conf.JWTSecretKey)

	if jwtSecretKey == "" {
		return nil, errors.Errorf("%s env variable is empty", conf.JWTSecretKey)
	}

	db, err := utils.ConnectDBFromEnv(DBEnvConf)
	if err != nil {
		return nil, errors.Wrap(err, "ConnectDBFromEnv failed")
	}

	amqpConnection, err := utils.ConnectAMQPFromEnv(AMQPEnvConf)
	if err != nil {
		return nil, errors.Wrap(err, "ConnectAMQPFromEnv failed")
	}

	channel, err := amqpConnection.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AMQP channel")
	}
	// This channel is only used to declare the exchange.
	defer utils.Close(channel)

	if err := utils.AMQPExchangeDeclare(channel, utils.SchedulerAMQPExchangeName); err != nil {
		return nil, errors.Wrap(err, "failed to declare AMQP exchange")
	}

	return &SchedulerServer{
		JWTSecretKey:   []byte(jwtSecretKey),
		DB:             db,
		AMQPConnection: amqpConnection,
		ConnectedNodes: make(map[uint64]chan *scheduler_proto.Task),
		AwaitingJobs:   make(map[uint64]chan *scheduler_proto.Task),
	}, nil
}

// Close closes the server resources.
func (s *SchedulerServer) Close() error {
	if err := s.DB.Close(); err != nil {
		return err
	}
	return s.AMQPConnection.Close()
}

// Serve creates a gRPC server and starts serving on a given address. This function is blocking and runs upon the server
// termination.
func Serve(grpcAddr string, srv *SchedulerServer) {
	utils.SetLogrusLevelFromEnv()

	// TODO: create the logrus entry in some parent scope, not here.
	logrusEntry := logrus.NewEntry(logrus.StandardLogger())
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	grpcServer := grpc.NewServer(
		grpc.Creds(utils.TLSServerCredentialsFromEnv(TLSEnvConf)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_logrus.UnaryServerInterceptor(logrusEntry),
			// grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry),
			grpc_auth.UnaryServerInterceptor(AuthFunc),
			grpc_validator.UnaryServerInterceptor(),
			middleware.StackTraceErrorInterceptor(false, LoggedErrorCodes...),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(AuthFunc)),
	)
	scheduler_proto.RegisterSchedulerServer(grpcServer, srv)

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
		if err := srv.HandleNodeConnectedEvents(); err != nil {
			f <- err
		}
	}(f)
	go func(f chan error) {
		if err := srv.HandleTaskStatusUpdatedEvents(); err != nil {
			f <- err
		}
	}(f)
	go func(f chan error) {
		if err := srv.HandleTaskCreatedEvents(); err != nil {
			f <- err
		}
	}(f)

	select {
	case <-c:
		logrus.Info("Gracefully stopping gRPC server...")
		close(srv.gracefulStop)
		grpcServer.GracefulStop()

	case err := <-f:
		logrus.Fatalf("GRPC server unexpectedly failed: %s", err.Error())
	}
}

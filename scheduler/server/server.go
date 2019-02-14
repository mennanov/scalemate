package server

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/jinzhu/gorm"
	"github.com/streadway/amqp"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
	// required by gorm
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

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

// SchedulerServer is a wrapper for `scheduler_proto.SchedulerServer` that holds the application specific settings
// and also implements some additional methods.
type SchedulerServer struct {
	// Logrus entry to be used for all the logging.
	logger *logrus.Logger
	DB     *gorm.DB
	// Authentication claims context injector.
	ClaimsInjector auth.ClaimsInjector
	// Events producer.
	Producer events.Producer
	// Events consumers.
	Consumers []events.Consumer
	// Tasks groupd by Node ID are sent to this channel as they are created by event listeners.
	NewTasksByNodeID map[uint64]chan *scheduler_proto.Task
	// Tasks grouped by Job ID are sent to this channel as they are created by event listeners.
	NewTasksByJobID map[uint64]chan *scheduler_proto.Task
	// gracefulStop channel is used to notify about the gRPC server GracefulStop() in progress. When the server is about
	// to stop this channel is closed, this will unblock all the receivers of this channel and they will receive a zero
	// value (empty struct) which should be discarded and take an appropriate action to gracefully finish long-running
	// request processing (streams, etc).
	// Requests that are expected to be processed quickly should not take any action as they will be waited to finish.
	gracefulStop chan struct{}
}

// Compile time interface check.
var _ scheduler_proto.SchedulerServer = new(SchedulerServer)

// WithLogger creates an option that sets the logger.
func WithLogger(logger *logrus.Logger) Option {
	return func(s *SchedulerServer) error {
		s.logger = logger
		return nil
	}
}

// WithDBConnection creates an option that sets the DB field to an existing DB connection.
func WithDBConnection(db *gorm.DB) Option {
	return func(s *SchedulerServer) error {
		s.DB = db
		return nil
	}
}

// WithClaimsInjector creates an option that sets ClaimsInjector field to a new auth.NewJWTClaimsInjector with JWT
// secret key from env.
func WithClaimsInjector(conf AppEnvConf) Option {
	return func(s *SchedulerServer) error {
		value := os.Getenv(conf.JWTSecretKey)
		if value == "" {
			return errors.New("JWT secret key is empty")
		}
		s.ClaimsInjector = auth.NewJWTClaimsInjector([]byte(value))
		return nil
	}
}

// WithAMQPProducer creates an option that sets the Producer field value to AMQPProducer.
func WithAMQPProducer(conn *amqp.Connection) Option {
	return func(s *SchedulerServer) error {
		if conn == nil {
			return errors.New("amqp.Connection is nil")
		}
		producer, err := events.NewAMQPProducer(conn, events.SchedulerAMQPExchangeName)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPProducer failed")
		}
		s.Producer = producer
		return nil
	}
}

// WithAMQPConsumers creates an option that sets the Consumers field value to AMQPConsumer(s).
func WithAMQPConsumers(conn *amqp.Connection) Option {
	return func(s *SchedulerServer) error {
		channel, err := conn.Channel()
		defer utils.Close(channel)

		if err != nil {
			return errors.Wrap(err, "failed to open a new AMQP channel")
		}
		// Declare all required exchanges.
		if err := events.AMQPExchangeDeclare(channel, events.SchedulerAMQPExchangeName); err != nil {
			return errors.Wrapf(err, "failed to declare AMQP exchange %s", events.SchedulerAMQPExchangeName)
		}

		jobStatusUpdatedToPendingConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_job_pending",
			"scheduler.job.updated.#.status.#",
			s.HandleJobPending)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for jobStatusUpdatedToPendingConsumer")
		}

		jobTerminatedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"",
			"scheduler.job.updated.#.status.#",
			s.HandleJobTerminated)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for jobTerminatedConsumer")
		}

		nodeConnectedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_node_connected",
			"scheduler.node.updated.#.connected_at.#",
			s.HandleNodeConnected)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for nodeConnectedConsumer")
		}

		nodeDisconnectedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_node_disconnected",
			"scheduler.node.updated.#.disconnected_at.#",
			s.HandleNodeDisconnected)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for nodeDisconnectedConsumer")
		}

		taskCreatedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"",
			"scheduler.task.created",
			s.HandleTaskCreated)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for taskCreatedConsumer")
		}

		taskTerminatedConsumer, err := events.NewAMQPConsumer(
			conn,
			events.SchedulerAMQPExchangeName,
			"scheduler_task_terminated",
			"scheduler.task.updated.#.status.#",
			s.HandleTaskTerminated)
		if err != nil {
			return errors.Wrap(err, "events.NewAMQPRawConsumer failed for taskTerminatedConsumer")
		}

		s.Consumers = []events.Consumer{
			jobStatusUpdatedToPendingConsumer,
			jobTerminatedConsumer,
			nodeConnectedConsumer,
			nodeDisconnectedConsumer,
			taskCreatedConsumer,
			taskTerminatedConsumer,
		}

		return nil
	}
}

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

// SchedulerEnvConf maps env variables for the Scheduler service.
var SchedulerEnvConf = AppEnvConf{
	BCryptCost:      "SCHEDULER_BCRYPT_COST",
	AccessTokenTTL:  "SCHEDULER_ACCESS_TOKEN_TTL",
	RefreshTokenTTL: "SCHEDULER_REFRESH_TOKEN_TTL",
	JWTSecretKey:    "SCHEDULER_JWT_SECRET_KEY",
}

// Option modifies the SchedulerServer.
type Option func(server *SchedulerServer) error

// NewSchedulerServer creates a new SchedulerServer and applies the given options to it.
func NewSchedulerServer(options ...Option) (*SchedulerServer, error) {
	s := &SchedulerServer{
		NewTasksByNodeID: make(map[uint64]chan *scheduler_proto.Task),
		NewTasksByJobID:  make(map[uint64]chan *scheduler_proto.Task),
		gracefulStop:     make(chan struct{}),
	}
	for _, option := range options {
		if err := option(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Close closes the server resources.
func (s *SchedulerServer) Close() error {
	for _, consumer := range s.Consumers {
		if err := consumer.Close(); err != nil {
			return errors.Wrap(err, "consumer.Close failed")
		}
	}

	return errors.Wrap(s.Producer.Close(), "SchedulerServer.Producer.Close failed")
}

var (
	// ErrNodeAlreadyConnected is used when the Node can't be "connected" because it is already connected.
	ErrNodeAlreadyConnected = errors.New("Node is already connected")
	// ErrNodeIsNotConnected is used when the Node can't be disconnected because it's already disconnected.
	ErrNodeIsNotConnected = errors.New("Node is not connected")
)

// ConnectNode adds the Node ID to the map of connected Nodes and updates the Node's status in DB.
// This method should be called when the Node connects to receive Tasks.
// FIXME: this method should be private.
func (s *SchedulerServer) ConnectNode(db *gorm.DB, node *models.Node) (*events_proto.Event, error) {
	if _, ok := s.NewTasksByNodeID[node.ID]; ok {
		return nil, ErrNodeAlreadyConnected
	}
	now := time.Now()
	event, err := node.Updates(db, map[string]interface{}{
		"connected_at": &now,
		"status":       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
	})
	if err != nil {
		return nil, errors.Wrap(err, "node.Updates failed")
	}
	s.NewTasksByNodeID[node.ID] = make(chan *scheduler_proto.Task)
	return event, nil
}

// DisconnectNode updates the Node status to OFFLINE.
// FIXME: this method should be private.
func (s *SchedulerServer) DisconnectNode(db *gorm.DB, node *models.Node) (*events_proto.Event, error) {
	if _, ok := s.NewTasksByNodeID[node.ID]; !ok {
		return nil, ErrNodeIsNotConnected
	}
	now := time.Now()
	nodeUpdatedEvent, err := node.Updates(db, map[string]interface{}{
		"disconnected_at": &now,
		"status":          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
	})
	if err != nil {
		return nil, errors.Wrap(err, "node.Updates failed")
	}

	close(s.NewTasksByNodeID[node.ID])
	delete(s.NewTasksByNodeID, node.ID)

	return nodeUpdatedEvent, nil
}

// Serve creates a gRPC server and starts serving on a given address.
// It also starts all the consumers.
func (s *SchedulerServer) Serve(grpcAddr string, shutdown chan os.Signal) {
	entry := logrus.NewEntry(s.logger)
	grpc_logrus.ReplaceGrpcLogger(entry)
	grpcServer := grpc.NewServer(
		grpc.Creds(utils.TLSServerCredentialsFromEnv(TLSEnvConf)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(entry),
			middleware.LoggerRequestIDInterceptor("request.id"),
			// grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry),
			grpc_auth.UnaryServerInterceptor(AuthFunc),
			grpc_validator.UnaryServerInterceptor(),
			middleware.StackTraceErrorInterceptor(false, LoggedErrorCodes...),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(AuthFunc)),
	)
	scheduler_proto.RegisterSchedulerServer(grpcServer, s)

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

	consumersCtx, consumersCtxCancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	defer func() {
		consumersCtxCancel()
		wg.Wait()
	}()

	for _, consumer := range s.Consumers {
		go consumer.Consume(consumersCtx, wg)
	}

	select {
	case <-shutdown:
		s.logger.Info("Gracefully stopping gRPC server...")
		close(s.gracefulStop)
		grpcServer.GracefulStop()

	case err := <-f:
		s.logger.WithError(err).Error("gRPC server unexpectedly failed")
	}
}

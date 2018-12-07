package server

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/jinzhu/gorm"
	"github.com/streadway/amqp"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
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
	DB *gorm.DB
	// Authentication claims context injector.
	ClaimsInjector auth.ClaimsInjector
	// utils.Publisher.
	Publisher utils.Publisher
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

// Connections represents third party services connections.
type Connections struct {
	DB   *gorm.DB
	AMQP *amqp.Connection
}

// NewSchedulerServerFromEnv creates a new SchedulerServer from environment variables.
// It also connects to the database and AMQP.
func NewSchedulerServerFromEnv(conf AppEnvConf) (*SchedulerServer, *Connections, error) {
	jwtSecretKey := os.Getenv(conf.JWTSecretKey)

	if jwtSecretKey == "" {
		return nil, nil, errors.Errorf("%s env variable is empty", conf.JWTSecretKey)
	}

	db, err := utils.ConnectDBFromEnv(DBEnvConf)
	if err != nil {
		return nil, nil, errors.Wrap(err, "ConnectDBFromEnv failed")
	}

	amqpConnection, err := utils.ConnectAMQPFromEnv(AMQPEnvConf)
	if err != nil {
		return nil, nil, errors.Wrap(err, "ConnectAMQPFromEnv failed")
	}

	publisher, err := utils.NewAMQPPublisher(amqpConnection, utils.SchedulerAMQPExchangeName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "utils.NewAMQPPublisher failed")
	}

	return &SchedulerServer{
		DB:               db,
		ClaimsInjector:   auth.NewJWTClaimsInjector([]byte(jwtSecretKey)),
		Publisher:        publisher,
		NewTasksByNodeID: make(map[uint64]chan *scheduler_proto.Task),
		NewTasksByJobID:  make(map[uint64]chan *scheduler_proto.Task),
	}, &Connections{DB: db, AMQP: amqpConnection}, nil
}

// Close closes the server resources.
func (s *SchedulerServer) Close() error {
	if err := s.DB.Close(); err != nil {
		return err
	}
	return s.Publisher.Close()
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
func (s *SchedulerServer) ConnectNode(node *models.Node) error {
	if _, ok := s.NewTasksByNodeID[node.ID]; ok {
		return ErrNodeAlreadyConnected
	}
	tx := s.DB.Begin()
	event, err := node.Updates(tx, map[string]interface{}{
		"connected_at": time.Now(),
		"status":       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
	})
	if err != nil {
		return err
	}
	if err := utils.SendAndCommit(tx, s.Publisher, event); err != nil {
		return err
	}
	s.NewTasksByNodeID[node.ID] = make(chan *scheduler_proto.Task)
	return nil
}

// DisconnectNode removes the Node ID from the map of connected Nodes and updates the Node's status in DB.
// It also updates the status of all the running Jobs and corresponding Tasks to NODE_FAILED.
// This method should be called when the Node disconnects.
// FIXME: this method should be private.
func (s *SchedulerServer) DisconnectNode(node *models.Node) error {
	if _, ok := s.NewTasksByNodeID[node.ID]; !ok {
		return ErrNodeIsNotConnected
	}
	tx := s.DB.Begin()
	var disconnectedNodeEvents []*events_proto.Event
	nodeUpdatedEvent, err := node.Updates(tx, map[string]interface{}{
		"disconnected_at": time.Now(),
		"status":          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
	})
	if err != nil {
		wrappedErr := errors.Wrap(err, "node.Updates failed")
		if txErr := utils.HandleDBError(tx.Rollback()); txErr != nil {
			return errors.Wrap(wrappedErr, errors.Wrap(txErr, "failed to rollback transaction").Error())
		}
		return wrappedErr
	}
	var tasks models.Tasks
	tasksEvents, err := tasks.UpdateStatusForDisconnectedNode(tx, node.ID)
	if err != nil {
		wrappedErr := errors.Wrap(err, "tasks.UpdateStatusForDisconnectedNode failed")
		if txErr := utils.HandleDBError(tx.Rollback()); txErr != nil {
			return errors.Wrap(wrappedErr, errors.Wrap(txErr, "failed to rollback transaction").Error())
		}
		return wrappedErr
	}
	jobIDs := make([]uint64, len(tasks))
	for i, task := range tasks {
		jobIDs[i] = task.JobID
	}
	var jobs models.Jobs
	jobsEvents, err := jobs.UpdateStatusForNodeFailedTasks(tx, jobIDs)
	if err != nil {
		wrappedErr := errors.Wrap(err, "jobs.UpdateStatusForNodeFailedTasks failed")
		if txErr := utils.HandleDBError(tx.Rollback()); txErr != nil {
			return errors.Wrap(wrappedErr, errors.Wrap(txErr, "failed to rollback transaction").Error())
		}
		return wrappedErr
	}
	disconnectedNodeEvents = append(disconnectedNodeEvents, nodeUpdatedEvent)
	disconnectedNodeEvents = append(disconnectedNodeEvents, tasksEvents...)
	disconnectedNodeEvents = append(disconnectedNodeEvents, jobsEvents...)
	if err := utils.SendAndCommit(tx, s.Publisher, disconnectedNodeEvents...); err != nil {
		return err
	}

	delete(s.NewTasksByNodeID, node.ID)
	return nil
}

// Serve creates a gRPC server and starts serving on a given address. This function is blocking and runs upon the server
// termination.
func Serve(ctx context.Context, grpcAddr string, srv *SchedulerServer) {
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

	select {
	case <-ctx.Done():
		logrus.Info("Gracefully stopping gRPC server...")
		close(srv.gracefulStop)
		grpcServer.GracefulStop()

	case err := <-f:
		logrus.WithError(err).Errorf("GRPC server unexpectedly failed")
	}
}

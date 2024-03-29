package server

import (
	"context"
	"net"
	"sync"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/jinzhu/gorm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	db     *gorm.DB
	// Authentication claims context injector.
	claimsInjector auth.ClaimsInjector
	// Events producer.
	producer events.Producer
	// Events consumers.
	consumers []events.Consumer
	// Tasks grouped by Node ID are sent to this channel as they are created by event listeners.
	tasksForNodes    map[uint64]chan *scheduler_proto.Task
	tasksForNodesMux *sync.RWMutex
	// Tasks grouped by Job ID are sent to this channel as they are created by event listeners.
	tasksForClients    map[uint64]chan *scheduler_proto.Task
	tasksForClientsMux *sync.RWMutex
	// gracefulStop channel is used to notify about the gRPC server GracefulStop() in progress. When the server is about
	// to stop this channel is closed, this will unblock all the receivers of this channel and they will receive a zero
	// value (empty struct) which should be discarded and take an appropriate action to gracefully finish long-running
	// request processing (streams, etc). Clients are expected to re-connect.
	// Requests that are expected to be processed quickly should not take any action as they will be waited to finish.
	gracefulStop chan struct{}
}

// Compile time interface check.
var _ scheduler_proto.SchedulerServer = new(SchedulerServer)

// NewSchedulerServer creates a new SchedulerServer and applies the given options to it.
func NewSchedulerServer(options ...Option) (*SchedulerServer, error) {
	s := &SchedulerServer{
		tasksForNodes:      make(map[uint64]chan *scheduler_proto.Task),
		tasksForClients:    make(map[uint64]chan *scheduler_proto.Task),
		gracefulStop:       make(chan struct{}),
		tasksForNodesMux:   new(sync.RWMutex),
		tasksForClientsMux: new(sync.RWMutex),
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
	for _, consumer := range s.consumers {
		if err := consumer.Close(); err != nil {
			return errors.Wrap(err, "consumer.Close failed")
		}
	}

	return errors.Wrap(s.producer.Close(), "SchedulerServer.producer.Close failed")
}

var (
	// ErrNodeAlreadyConnected is used when the Node can't be "connected" because it is already connected.
	ErrNodeAlreadyConnected = status.Error(codes.FailedPrecondition, "Node is already connected")
)

// Serve creates a gRPC server and starts serving on a given address.
// It also starts all the consumers.
// The service is gracefully stopped when the given context is Done.
func (s *SchedulerServer) Serve(ctx context.Context, grpcAddr string) {
	entry := logrus.NewEntry(s.logger)
	grpc_logrus.ReplaceGrpcLogger(entry)
	grpcServer := grpc.NewServer(
		grpc.Creds(utils.NewServerTLSCredentialsFromFile(TLSEnvConf)),
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

	for _, consumer := range s.consumers {
		go consumer.Consume(ctx)
	}

	select {
	case <-ctx.Done():
		s.logger.Info("Gracefully stopping gRPC server...")
		close(s.gracefulStop)
		grpcServer.GracefulStop()

	case err := <-f:
		s.logger.WithError(err).Error("gRPC server unexpectedly failed")
	}
	s.logger.Info("gRPC server is stopped.")
}

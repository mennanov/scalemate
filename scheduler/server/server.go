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
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

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

// SchedulerServer is a wrapper for `scheduler_proto.SchedulerServer` that holds the application specific settings
// and also implements some additional methods.
type SchedulerServer struct {
	// Logrus entry to be used for all the logging.
	logger *logrus.Logger
	db     *sqlx.DB
	// Authentication claims context injector.
	claimsInjector auth.ClaimsInjector
	// Events producer.
	producer events.Producer
	// Containers grouped by Node ID are sent to this channel as they are scheduled by event handlers.
	// It is used to send scheduled containers to their connected Nodes.
	// When the Node connects the corresponding channel is created, otherwise it is nil.
	containersByNodeId   map[int64]chan *scheduler_proto.Container
	containersByNodeIdMu *sync.RWMutex
	// Container updates grouped by container ID are sent to this channel (that also includes Limits updates).
	containerUpdatesById   map[int64]chan *scheduler_proto.ContainerUpdate
	containerUpdatesByIdMu *sync.RWMutex
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
func NewSchedulerServer(options ...Option) *SchedulerServer {
	s := &SchedulerServer{
		containersByNodeId:     make(map[int64]chan *scheduler_proto.Container),
		containersByNodeIdMu:   new(sync.RWMutex),
		containerUpdatesById:   make(map[int64]chan *scheduler_proto.ContainerUpdate),
		containerUpdatesByIdMu: new(sync.RWMutex),
		gracefulStop:           make(chan struct{}),
	}
	for _, option := range options {
		option(s)
	}

	return s
}

// Serve creates a gRPC server and starts serving on a given address.
// It also starts all the consumers.
// The service is gracefully stopped when the given context is Done.
func (s *SchedulerServer) Serve(ctx context.Context, grpcAddr string, creds credentials.TransportCredentials) {
	entry := logrus.NewEntry(s.logger)
	grpc_logrus.ReplaceGrpcLogger(entry)
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_logrus.UnaryServerInterceptor(entry),
			middleware.LoggerRequestIDInterceptor("request.id"),
			grpc_auth.UnaryServerInterceptor(AuthFunc),
			middleware.StackTraceErrorInterceptor(false, LoggedErrorCodes...),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(AuthFunc)),
	)
	scheduler_proto.RegisterSchedulerServer(grpcServer, s)

	grpcErrors := make(chan error, 1)
	go func(f chan error) {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			f <- err
			return
		}
		s.logger.Infof("Serving on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			f <- err
		}
	}(grpcErrors)

	select {
	case <-ctx.Done():
		s.logger.Info("Gracefully stopping gRPC server...")
		close(s.gracefulStop)
		grpcServer.GracefulStop()

	case err := <-grpcErrors:
		s.logger.WithError(err).Error("gRPC server unexpectedly failed")
	}
	s.logger.Info("gRPC server is stopped.")
}

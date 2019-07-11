package backend

import (
	"context"
	"net"
	"sync"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/middleware"
)

// SchedulerBackend is a wrapper for `scheduler_proto.SchedulerBackend` that holds the application specific settings
// and also implements some additional methods.
type SchedulerBackend struct {
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
	// gracefulStop channel is used to notify about the gRPC server GracefulStop() in progress. When the server is about
	// to stop this channel is closed, this will unblock all the receivers of this channel and they will receive a zero
	// value (empty struct) which should be discarded and take an appropriate action to gracefully finish long-running
	// request processing (streams, etc). Clients are expected to re-connect.
	// Requests that are expected to be processed quickly should not take any action as they will be waited to finish.
	gracefulStop chan struct{}
}

// Compile time interface check.
var _ scheduler_proto.SchedulerBackEndServer = new(SchedulerBackend)

// NewSchedulerBackend creates a new SchedulerBackend and applies the given options to it.
func NewSchedulerBackend(options ...Option) *SchedulerBackend {
	s := &SchedulerBackend{
		containersByNodeId:     make(map[int64]chan *scheduler_proto.Container),
		containersByNodeIdMu:   new(sync.RWMutex),
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
// TODO: reduce code duplication (see frontend/server.go).
func (s *SchedulerBackend) Serve(ctx context.Context, grpcAddr string, creds credentials.TransportCredentials) {
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			middleware.RequestsLoggerInterceptor(s.logger),
			middleware.ClaimsInjectorUnaryInterceptor(s.claimsInjector),
			grpc_recovery.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(middleware.ClaimsInjectorStreamingInterceptor(s.claimsInjector)),
	)
	scheduler_proto.RegisterSchedulerBackEndServer(grpcServer, s)

	grpcErrors := make(chan error, 1)
	go func(f chan error) {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			f <- err
			return
		}
		s.logger.Infof("Serving SchedulerBackend on %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			f <- err
		}
	}(grpcErrors)

	select {
	case <-ctx.Done():
		s.logger.Info("Gracefully stopping SchedulerBackend gRPC server...")
		close(s.gracefulStop)
		grpcServer.GracefulStop()

	case err := <-grpcErrors:
		s.logger.WithError(err).Error("SchedulerBackend gRPC server unexpectedly failed")
	}
	s.logger.Info("SchedulerBackend gRPC server is stopped.")
}

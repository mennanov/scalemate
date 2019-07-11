package client

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// newGRPCConnection creates a new connection to a gRPC service.
func newGRPCConnection(addr string) *grpc.ClientConn {
	e := logrus.NewEntry(logrus.StandardLogger())
	// TODO: update this part to use a real certificate.
	creds, err := credentials.NewClientTLSFromFile("cert/cert.pem", "localhost")
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithDialer(func(addr string, d time.Duration) (net.Conn, error) {
			return net.DialTimeout("tcp", addr, d)
		}),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(e)),
	)

	if err != nil {
		panic(errors.Wrap(err, "could not connect to a gRPC service"))
	}

	return conn
}

// NewAccountsClient create a new AccountsClient instance.
func NewAccountsClient(addr string) accounts_proto.AccountsClient {
	conn := newGRPCConnection(fmt.Sprintf("dns://%s", addr))
	return accounts_proto.NewAccountsClient(conn)
}

// NewSchedulerFrontEndClient creates a new SchedulerClient instance.
func NewSchedulerFrontEndClient(addr string) scheduler_proto.SchedulerFrontEndClient {
	conn := newGRPCConnection(fmt.Sprintf("dns://%s", addr))
	return scheduler_proto.NewSchedulerFrontEndClient(conn)
}

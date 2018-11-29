package accounts

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ServiceAddr is the Accounts gRPC service net address, e.g. "localhost:8000".
var ServiceAddr string

// NewConn creates a new connection to the Accounts gRPC service.
func NewConn(addr string) *grpc.ClientConn {
	e := logrus.NewEntry(logrus.StandardLogger())
	// TODO: update this part to use real certificate.
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
			return net.DialTimeout("tcp", ServiceAddr, d)
		}),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(e)),
	)

	if err != nil {
		panic(errors.Wrap(err, "could not connect to accounts service"))
	}

	return conn
}

// NewAccountsClient create a new AccountsClient instance.
func NewAccountsClient() accounts_proto.AccountsClient {
	conn := NewConn(fmt.Sprintf("dns://%s", ServiceAddr))
	return accounts_proto.NewAccountsClient(conn)
}

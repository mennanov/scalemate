package server_test

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"gopkg.in/khaiql/dbcleaner.v2"
	"gopkg.in/khaiql/dbcleaner.v2/engine"

	"github.com/mennanov/scalemate/scheduler/event_listeners"
	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

var verbose bool
var cleaner = dbcleaner.New()

func init() {
	flag.BoolVar(&verbose, "verbose", false, "verbose")
	flag.Parse()
}

type ServerTestSuite struct {
	suite.Suite
	service        *server.SchedulerServer
	client         scheduler_proto.SchedulerClient
	amqpConnection *amqp.Connection
}

func (s *ServerTestSuite) SetupSuite() {
	// Start gRPC server.
	localAddr := "localhost:50052"

	db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
	s.Require().NoError(err)

	amqpConnection, err := utils.ConnectAMQPFromEnv(server.AMQPEnvConf)
	s.Require().NoError(err)
	s.amqpConnection = amqpConnection

	publisher, err := utils.NewAMQPPublisher(amqpConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	s.service = &server.SchedulerServer{
		DB: db,
		ClaimsInjector: auth.NewFakeClaimsContextInjector(&auth.Claims{
			Username:  "test_username",
			Role:      accounts_proto.User_USER,
			TokenType: "access",
		}),
		Publisher:        publisher,
		NewTasksByNodeID: make(map[uint64]chan *scheduler_proto.Task),
		NewTasksByJobID:  make(map[uint64]chan *scheduler_proto.Task),
	}

	// Start event listeners.
	preparedListeners, err := event_listeners.SetUpAMQPEventListeners(event_listeners.RegisteredEventListeners, amqpConnection)
	s.Require().NoError(err)
	wg := &sync.WaitGroup{}
	event_listeners.RunEventListeners(context.Background(), wg, preparedListeners, s.service)

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.service.DB))

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv(server.DBEnvConf.Host), os.Getenv(server.DBEnvConf.Port), os.Getenv(server.DBEnvConf.User),
		os.Getenv(server.DBEnvConf.Name), os.Getenv(server.DBEnvConf.Password))
	pg := engine.NewPostgresEngine(dsn)
	cleaner.SetEngine(pg)

	go func() {
		server.Serve(context.Background(), localAddr, s.service)
	}()

	// Waiting for GRPC server to start serving.
	time.Sleep(time.Millisecond * 100)
	s.client = scheduler_proto.NewSchedulerClient(newTestConn(localAddr))
}

func (s *ServerTestSuite) SetupTest() {
	for _, tableName := range models.TableNames {
		cleaner.Acquire(tableName)
	}
}

func (s *ServerTestSuite) TearDownTest() {
	for _, tableName := range models.TableNames {
		cleaner.Clean(tableName)
	}
}

//func (s *ServerTestSuite) TearDownSuite() {
//	migrations.RollbackAllMigrations(s.service.DB)
//}

func TestRunServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func newTestConn(addr string) *grpc.ClientConn {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(tlsClientCredentialsFromEnv()),
		grpc.WithDialer(func(addr string, d time.Duration) (net.Conn, error) {
			return net.Dial("tcp", addr)
		}),
	)

	if err != nil {
		panic(err)
	}

	return conn
}

func tlsClientCredentialsFromEnv() credentials.TransportCredentials {
	certFile := os.Getenv(server.TLSEnvConf.CertFile)

	if certFile == "" {
		panic(fmt.Sprintf("%s env variable is empty", server.TLSEnvConf.CertFile))
	}

	// Bazel specific path to data files. See https://docs.bazel.build/versions/master/build-ref.html#data
	srcDir := os.Getenv("TEST_SRCDIR")
	if srcDir != "" {
		certFile = path.Join(srcDir, certFile)
	}

	creds, err := credentials.NewClientTLSFromFile(certFile, "localhost")
	if err != nil {
		panic(err)
	}

	return creds
}

// FIXME: move to testutils.
func (s *ServerTestSuite) assertGRPCError(err error, code codes.Code) {
	s.Require().Error(err)
	if statusCode, ok := status.FromError(err); ok {
		if !s.Equal(code, statusCode.Code()) {
			fmt.Println("Error message: ", statusCode.Message())
		}
	} else {
		s.Fail("Unknown type of returned error")
	}
}

package server_test

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/suite"
	"gopkg.in/khaiql/dbcleaner.v2"

	"google.golang.org/grpc"

	"github.com/google/uuid"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"gopkg.in/khaiql/dbcleaner.v2/engine"

	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/auth"
)

var verbose bool
var cleaner = dbcleaner.New()

func init() {
	flag.BoolVar(&verbose, "verbose", false, "verbose")
	flag.Parse()
}

type ServerTestSuite struct {
	suite.Suite
	service *server.SchedulerServer
	client  scheduler_proto.SchedulerClient
}

func (s *ServerTestSuite) SetupSuite() {
	// Start gRPC server.
	localAddr := "localhost:50052"
	var err error
	s.service, err = server.NewSchedulerServerFromEnv(server.SchedulerEnvConf)

	s.Require().NoError(err)

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.service.DB))

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv(server.DBEnvConf.Host), os.Getenv(server.DBEnvConf.Port), os.Getenv(server.DBEnvConf.User),
		os.Getenv(server.DBEnvConf.Name), os.Getenv(server.DBEnvConf.Password))
	pg := engine.NewPostgresEngine(dsn)
	cleaner.SetEngine(pg)

	if err != nil {
		panic(fmt.Sprintf("Failed to create gRPC server: %+v", err))
	}
	go func() {
		server.Serve(localAddr, s.service)
	}()

	// Waiting for GRPC server to start serving.
	time.Sleep(time.Millisecond * 100)
	s.client = scheduler_proto.NewSchedulerClient(newTestConn(localAddr))
}

func (s *ServerTestSuite) SetupTest() {
	cleaner.Acquire("nodes")
	cleaner.Acquire("tasks")
	cleaner.Acquire("jobs")
}

func (s *ServerTestSuite) TearDownTest() {
	cleaner.Clean("nodes")
	cleaner.Clean("tasks")
	cleaner.Clean("jobs")
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

func (s *ServerTestSuite) createToken(username, nodeName string, role accounts_proto.User_Role, tokenType string, ttl time.Duration) string {
	now := time.Now()
	expiresAt := now.Add(ttl).Unix()

	claims := &auth.Claims{
		Username:  username,
		NodeName:  nodeName,
		Role:      role,
		TokenType: tokenType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
			Issuer:    "Scalemate.io",
			IssuedAt:  now.Unix(),
			Id:        uuid.New().String(),
		},
	}
	tokenString, err := claims.SignedString(s.service.JWTSecretKey)
	if err != nil {
		panic(err)
	}
	return tokenString
}

func tokensFakeSaver(*accounts_proto.AuthTokens) error {
	return nil
}

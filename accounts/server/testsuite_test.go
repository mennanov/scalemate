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

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	"github.com/mennanov/scalemate/accounts/migrations"
	"github.com/mennanov/scalemate/shared/utils"

	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"gopkg.in/khaiql/dbcleaner.v2"
	"gopkg.in/khaiql/dbcleaner.v2/engine"
	"gopkg.in/khaiql/dbcleaner.v2/logging"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/auth"
)

var verbose bool
var cleaner = dbcleaner.New(dbcleaner.SetLogger(&logging.Stdout{}))

func init() {
	flag.BoolVar(&verbose, "verbose", false, "verbose")
	flag.Parse()
}

type ServerTestSuite struct {
	suite.Suite
	service         *server.AccountsServer
	client          accounts_proto.AccountsClient
	db              *gorm.DB
	amqpChannel     *amqp.Channel
	amqpConnection  *amqp.Connection
	ctxCancel       context.CancelFunc
	bCryptCost      int
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	jwtSecretKey    []byte
}

func (s *ServerTestSuite) SetupSuite() {
	// Start gRPC server.
	localAddr := "localhost:50051"
	var err error

	s.amqpConnection, err = utils.ConnectAMQPFromEnv(server.AMQPEnvConf)
	s.Require().NoError(err)

	s.db, err = utils.ConnectDBFromEnv(server.DBEnvConf)
	s.Require().NoError(err)

	s.jwtSecretKey, err = server.JWTSecretKeyFromEnv(server.AccountsEnvConf)
	s.Require().NoError(err)

	s.bCryptCost, err = server.BCryptCostFromEnv(server.AccountsEnvConf)
	s.Require().NoError(err)

	s.accessTokenTTL, err = server.AccessTokenFromEnv(server.AccountsEnvConf)
	s.Require().NoError(err)

	s.refreshTokenTTL, err = server.RefreshTokenFromEnv(server.AccountsEnvConf)
	s.Require().NoError(err)

	s.service, err = server.NewAccountsServer(
		server.WithLogger(logrus.New()),
		server.WithDBConnection(s.db),
		server.WithAMQPConsumers(s.amqpConnection),
		server.WithAMQPProducer(s.amqpConnection),
		server.WithJWTSecretKey(s.jwtSecretKey),
		server.WithClaimsInjector(s.jwtSecretKey),
		server.WithAccessTokenTTL(s.accessTokenTTL),
		server.WithRefreshTokenTTL(s.refreshTokenTTL),
		server.WithBCryptCost(s.bCryptCost),
	)
	s.Require().NoError(err)

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.db))

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv(server.DBEnvConf.Host), os.Getenv(server.DBEnvConf.Port), os.Getenv(server.DBEnvConf.User),
		os.Getenv(server.DBEnvConf.Name), os.Getenv(server.DBEnvConf.Password))
	pg := engine.NewPostgresEngine(dsn)
	cleaner.SetEngine(pg)

	if err != nil {
		panic(fmt.Sprintf("Failed to create gRPC server: %+v", err))
	}

	var ctx context.Context
	ctx, s.ctxCancel = context.WithCancel(context.Background())

	go func() {
		s.service.Serve(ctx, localAddr)
		defer utils.Close(s.service)
	}()

	// Waiting for gRPC server to start serving.
	time.Sleep(time.Millisecond * 100)
	s.client = accounts_proto.NewAccountsClient(newTestConn(localAddr))
}

func (s *ServerTestSuite) SetupTest() {
	var err error
	s.amqpChannel, err = s.amqpConnection.Channel()
	s.Require().NoError(err)

	cleaner.Acquire("users")
	cleaner.Acquire("nodes")
}

func (s *ServerTestSuite) TearDownTest() {
	s.Require().NoError(s.amqpChannel.Close())

	cleaner.Clean("users")
	cleaner.Clean("nodes")
}

func (s *ServerTestSuite) TearDownSuite() {
	s.ctxCancel()
	// Wait for the service to stop gracefully.
	time.Sleep(time.Millisecond * 200)
	s.NoError(migrations.RollbackAllMigrations(s.db))
}

func TestRunServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func newTestConn(addr string) *grpc.ClientConn {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(tlsClientCredentialsFromEnv(server.TLSEnvConf)),
		grpc.WithDialer(func(addr string, d time.Duration) (net.Conn, error) {
			return net.Dial("tcp", addr)
		}),
	)

	if err != nil {
		panic(err)
	}

	return conn
}

func tlsClientCredentialsFromEnv(conf utils.TLSEnvConf) credentials.TransportCredentials {
	certFile := os.Getenv(conf.CertFile)

	if certFile == "" {
		panic(fmt.Sprintf("%s env variable is empty", conf.CertFile))
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

func (s *ServerTestSuite) createTestUserQuick(password string) *models.User {
	user := &models.User{
		Username: "username",
		Email:    "valid@email.com",
		Role:     accounts_proto.User_USER,
		Banned:   false,
	}
	s.Require().NoError(user.SetPasswordHash(password, s.bCryptCost))
	s.db.Create(user)
	return user
}

func (s *ServerTestSuite) createTestUser(user *models.User, password string) *models.User {
	s.Require().NoError(user.SetPasswordHash(password, s.bCryptCost))
	s.db.Create(user)
	return user
}

func (s *ServerTestSuite) assertGRPCError(err error, code codes.Code) {
	s.Require().Error(err)
	if statusCode, ok := status.FromError(err); ok {
		if !s.Equal(code, statusCode.Code()) {
			fmt.Println("Error message: ", statusCode.Message())
		}
	} else {
		s.Fail(fmt.Sprintf("Unknown type of returned error: %T: %s", err, err.Error()))
	}
}

func (s *ServerTestSuite) assertAuthTokensValid(tokens *accounts_proto.AuthTokens, user *models.User, issuedAt time.Time, nodeName string) {
	accessTokenClaims, err := auth.NewClaimsFromStringVerified(tokens.AccessToken, s.jwtSecretKey)
	s.Require().NoError(err)

	s.Equal(user.Username, accessTokenClaims.Username)
	s.Equal(user.Role, accessTokenClaims.Role)
	s.Equal(issuedAt.Unix(), accessTokenClaims.IssuedAt)
	s.Equal(auth.TokenTypeAccess, accessTokenClaims.TokenType)
	s.WithinDuration(issuedAt.Add(s.accessTokenTTL), time.Unix(accessTokenClaims.ExpiresAt, 0), s.accessTokenTTL)
	s.Equal(nodeName, accessTokenClaims.NodeName)

	refreshTokenClaims, err := auth.NewClaimsFromStringVerified(tokens.RefreshToken, s.jwtSecretKey)
	s.Require().NoError(err)

	s.Equal(user.Username, refreshTokenClaims.Username)
	s.Equal(user.Role, refreshTokenClaims.Role)
	s.Equal(issuedAt.Unix(), refreshTokenClaims.IssuedAt)
	s.Equal(auth.TokenTypeRefresh, refreshTokenClaims.TokenType)
	s.WithinDuration(issuedAt.Add(s.refreshTokenTTL), time.Unix(refreshTokenClaims.ExpiresAt, 0), s.refreshTokenTTL)
	s.Equal(nodeName, refreshTokenClaims.NodeName)

	s.True(accessTokenClaims.ExpiresAt < refreshTokenClaims.ExpiresAt)
	s.NotEqual(accessTokenClaims.Id, refreshTokenClaims.Id)
}

func (s *ServerTestSuite) accessCredentialsQuick(ttl time.Duration, role accounts_proto.User_Role) grpc.CallOption {
	adminUser := &models.User{
		Username: "access_username",
		Email:    "access@scalemate.io",
		Role:     role,
		Banned:   false,
	}

	return s.userAccessCredentials(adminUser, ttl)
}

func (s *ServerTestSuite) userAccessCredentials(user *models.User, ttl time.Duration) grpc.CallOption {
	accessToken, err := user.NewJWTSigned(ttl, auth.TokenTypeAccess, s.jwtSecretKey, "")
	s.Require().NoError(err)
	refreshToken, err := user.NewJWTSigned(ttl*2, auth.TokenTypeRefresh, s.jwtSecretKey, "")
	s.Require().NoError(err)
	jwtCredentials := auth.NewJWTCredentials(
		s.client,
		&accounts_proto.AuthTokens{AccessToken: accessToken, RefreshToken: refreshToken},
		func(tokens *accounts_proto.AuthTokens) error { return nil })
	return grpc.PerRPCCredentials(jwtCredentials)
}

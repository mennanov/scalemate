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

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/accounts/server"
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
	service *server.AccountsServer
	client  accounts_proto.AccountsClient
}

func (s *ServerTestSuite) SetupSuite() {
	// Start gRPC server.
	localAddr := "localhost:50051"
	var err error
	s.service, err = server.NewAccountServerFromEnv(server.AccountsEnvConf)

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
	s.client = accounts_proto.NewAccountsClient(newTestConn(localAddr))
}

func (s *ServerTestSuite) SetupTest() {
	cleaner.Acquire("users")
	cleaner.Acquire("nodes")
}

func (s *ServerTestSuite) TearDownTest() {
	cleaner.Clean("users")
	cleaner.Clean("nodes")
}

func (s *ServerTestSuite) TearDownSuite() {
	migrations.RollbackAllMigrations(s.service.DB)
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
	user.SetPasswordHash(password, s.service.BcryptCost)
	s.service.DB.Create(user)
	return user
}

func (s *ServerTestSuite) createTestUser(user *models.User, password string) *models.User {
	user.SetPasswordHash(password, s.service.BcryptCost)
	s.service.DB.Create(user)
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
	accessTokenClaims, err := auth.NewClaimsFromStringVerified(tokens.AccessToken, s.service.JWTSecretKey)
	s.Require().NoError(err)

	s.Equal(user.Username, accessTokenClaims.Username)
	s.Equal(user.Role, accessTokenClaims.Role)
	s.Equal(issuedAt.Unix(), accessTokenClaims.IssuedAt)
	s.Equal("access", accessTokenClaims.TokenType)
	s.WithinDuration(issuedAt.Add(s.service.AccessTokenTTL), time.Unix(accessTokenClaims.ExpiresAt, 0), s.service.AccessTokenTTL)
	s.Equal(nodeName, accessTokenClaims.NodeName)

	refreshTokenClaims, err := auth.NewClaimsFromStringVerified(tokens.RefreshToken, s.service.JWTSecretKey)
	s.Require().NoError(err)

	s.Equal(user.Username, refreshTokenClaims.Username)
	s.Equal(user.Role, refreshTokenClaims.Role)
	s.Equal(issuedAt.Unix(), refreshTokenClaims.IssuedAt)
	s.Equal("refresh", refreshTokenClaims.TokenType)
	s.WithinDuration(issuedAt.Add(s.service.RefreshTokenTTL), time.Unix(refreshTokenClaims.ExpiresAt, 0), s.service.RefreshTokenTTL)
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
	accessToken, err := user.NewJWTSigned(ttl, true, s.service.JWTSecretKey, "")
	s.Require().NoError(err)
	refreshToken, err := user.NewJWTSigned(ttl*2, false, s.service.JWTSecretKey, "")
	s.Require().NoError(err)
	jwtCredentials := auth.NewJWTCredentials(
		s.client,
		&accounts_proto.AuthTokens{AccessToken: accessToken, RefreshToken: refreshToken},
		func(tokens *accounts_proto.AuthTokens) error { return nil })
	return grpc.PerRPCCredentials(jwtCredentials)
}

package server_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"

	"github.com/mennanov/scalemate/accounts/conf"
	"github.com/mennanov/scalemate/accounts/migrations"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"

	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/auth"
)

const natsDurableName = "accounts-server-tests"

type ServerTestSuite struct {
	suite.Suite
	service         *server.AccountsServer
	client          accounts_proto.AccountsClient
	db              *gorm.DB
	sc              stan.Conn
	subscription    events.Subscription
	messagesHandler *events.MessagesTestingHandler
	ctxCancel       context.CancelFunc
	logger          *logrus.Logger
}

func (s *ServerTestSuite) SetupSuite() {
	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	// Start gRPC server.
	localAddr := "localhost:50051"
	var err error

	s.db, err = utils.CreateTestingDatabase(conf.AccountsConf.DBUrl, "accounts_server_test_suite")
	s.Require().NoError(err)

	s.sc, err = stan.Connect(
		conf.AccountsConf.NatsClusterName,
		fmt.Sprintf("accounts-service-%s", uuid.New().String()),
		stan.NatsURL(conf.AccountsConf.NatsAddr))
	s.Require().NoError(err)

	consumer := events.NewNatsConsumer(s.sc, events.AccountsSubjectName, s.logger, stan.DurableName(natsDurableName))
	s.messagesHandler = &events.MessagesTestingHandler{}
	s.subscription, err = consumer.Consume(s.messagesHandler)

	s.service, err = server.NewAccountsServer(
		server.WithLogger(logrus.New()),
		server.WithDBConnection(s.db),
		server.WithProducer(events.NewNatsProducer(s.sc, events.AccountsSubjectName)),
		server.WithJWTSecretKey(conf.AccountsConf.JWTSecretKey),
		server.WithClaimsInjector(auth.NewJWTClaimsInjector(conf.AccountsConf.JWTSecretKey)),
		server.WithAccessTokenTTL(conf.AccountsConf.AccessTokenTTL),
		server.WithRefreshTokenTTL(conf.AccountsConf.RefreshTokenTTL),
		server.WithBCryptCost(conf.AccountsConf.BCryptCost),
	)
	s.Require().NoError(err)

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.db))

	var ctx context.Context
	ctx, s.ctxCancel = context.WithCancel(context.Background())

	serverCreds, err := credentials.NewServerTLSFromFile(conf.AccountsConf.TLSCertFile, conf.AccountsConf.TLSKeyFile)
	s.Require().NoError(err)

	go func() {
		s.service.Serve(ctx, localAddr, serverCreds)
	}()

	clientCreds, err := credentials.NewClientTLSFromFile(conf.AccountsConf.TLSCertFile, "localhost")
	s.Require().NoError(err)
	s.client = accounts_proto.NewAccountsClient(newClientConn(localAddr, clientCreds))
}

func (s *ServerTestSuite) TearDownTest() {
	s.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", strings.Join(models.TableNames, ",")))
}

func (s *ServerTestSuite) TearDownSuite() {
	s.ctxCancel()
	// Wait for the service to stop gracefully.
	time.Sleep(time.Millisecond * 100)

	utils.Close(s.subscription, s.logger)
	utils.Close(s.sc, s.logger)
	utils.Close(s.db, s.logger)
}

func TestRunServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func newClientConn(addr string, creds credentials.TransportCredentials) *grpc.ClientConn {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		panic(err)
	}

	return conn
}

func (s *ServerTestSuite) createTestUserQuick(password string) *models.User {
	user := &models.User{
		Username: "username",
		Email:    "valid@email.com",
		Role:     accounts_proto.User_USER,
		Banned:   false,
	}
	s.Require().NoError(user.SetPasswordHash(password, conf.AccountsConf.BCryptCost))
	s.db.Create(user)
	return user
}

func (s *ServerTestSuite) createTestUser(user *models.User, password string) *models.User {
	s.Require().NoError(user.SetPasswordHash(password, conf.AccountsConf.BCryptCost))
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
	accessTokenClaims, err := auth.NewClaimsFromStringVerified(tokens.AccessToken, conf.AccountsConf.JWTSecretKey)
	s.Require().NoError(err)

	s.Equal(user.Username, accessTokenClaims.Username)
	s.Equal(user.Role, accessTokenClaims.Role)
	s.Equal(issuedAt.Unix(), accessTokenClaims.IssuedAt)
	s.Equal(auth.TokenTypeAccess, accessTokenClaims.TokenType)
	s.WithinDuration(issuedAt.Add(conf.AccountsConf.AccessTokenTTL), time.Unix(accessTokenClaims.ExpiresAt, 0),
		conf.AccountsConf.AccessTokenTTL)
	s.Equal(nodeName, accessTokenClaims.NodeName)

	refreshTokenClaims, err := auth.NewClaimsFromStringVerified(tokens.RefreshToken, conf.AccountsConf.JWTSecretKey)
	s.Require().NoError(err)

	s.Equal(user.Username, refreshTokenClaims.Username)
	s.Equal(user.Role, refreshTokenClaims.Role)
	s.Equal(issuedAt.Unix(), refreshTokenClaims.IssuedAt)
	s.Equal(auth.TokenTypeRefresh, refreshTokenClaims.TokenType)
	s.WithinDuration(issuedAt.Add(conf.AccountsConf.RefreshTokenTTL), time.Unix(refreshTokenClaims.ExpiresAt, 0),
		conf.AccountsConf.RefreshTokenTTL)
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
	accessToken, err := user.NewJWTSigned(ttl, auth.TokenTypeAccess, conf.AccountsConf.JWTSecretKey, "")
	s.Require().NoError(err)
	refreshToken, err := user.NewJWTSigned(ttl*2, auth.TokenTypeRefresh, conf.AccountsConf.JWTSecretKey, "")
	s.Require().NoError(err)
	jwtCredentials := auth.NewJWTCredentials(
		s.client,
		&accounts_proto.AuthTokens{AccessToken: accessToken, RefreshToken: refreshToken},
		func(tokens *accounts_proto.AuthTokens) error { return nil })
	return grpc.PerRPCCredentials(jwtCredentials)
}

package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // keep
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"

	"github.com/mennanov/scalemate/accounts/conf"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"

	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/auth"
)

const (
	natsDurableName = "accounts-server-tests"
	dbName          = "accounts_server_test_suite"
)

type ServerTestSuite struct {
	suite.Suite
	service         *server.AccountsServer
	client          accounts_proto.AccountsClient
	db              *sqlx.DB
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

	db, err := testutils.CreateTestingDatabase(conf.AccountsConf.DBUrl, dbName)
	s.Require().NoError(err)
	s.db = db

	s.sc, err = stan.Connect(
		conf.AccountsConf.NatsClusterName,
		fmt.Sprintf("accounts-service-%s", uuid.New().String()),
		stan.NatsURL(conf.AccountsConf.NatsAddr))
	s.Require().NoError(err)

	consumer := events.NewNatsConsumer(s.sc, events.AccountsSubjectName, s.logger, stan.DurableName(natsDurableName))
	s.messagesHandler = events.NewMessagesTestingHandler()
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

	// Run migrations.
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{
		MigrationsTable: "migrations",
		DatabaseName:    dbName,
		SchemaName:      "",
	})
	m, err := migrate.NewWithDatabaseInstance("file://../migrations", dbName, driver)
	s.Require().NoError(err)
	s.Require().NoError(m.Up())

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

func (s *ServerTestSuite) SetupTest() {
	testutils.TruncateTables(s.db)
}

func (s *ServerTestSuite) TearDownSuite() {
	s.ctxCancel()

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

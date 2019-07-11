package server_test

import (
	"context"
	"fmt"
	"testing"

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

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/auth"
)

const (
	dbName = "accounts_server_test_suite"
)

type ServerTestSuite struct {
	suite.Suite
	service         *server.AccountsServer
	client          accounts_proto.AccountsClient
	db              *sqlx.DB
	sc              stan.Conn
	messagesHandler *testutils.ExpectMessagesHandler
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

	producer := events.NewNatsProducer(s.sc, events.AccountsSubjectName, 5)

	s.service, err = server.NewAccountsServer(
		server.WithLogger(logrus.New()),
		server.WithDBConnection(s.db),
		server.WithProducer(producer),
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

	serverCreds, err := credentials.NewServerTLSFromFile(conf.AccountsConf.TLSCertFile, conf.AccountsConf.TLSKeyFile)
	s.Require().NoError(err)

	var ctx context.Context
	ctx, s.ctxCancel = context.WithCancel(context.Background())

	go func() {
		s.service.Serve(ctx, localAddr, serverCreds)
	}()

	clientCreds, err := credentials.NewClientTLSFromFile(conf.AccountsConf.TLSCertFile, "localhost")
	s.Require().NoError(err)
	s.client = accounts_proto.NewAccountsClient(testutils.NewClientConn(localAddr, clientCreds))
}

func (s *ServerTestSuite) SetupTest() {
	testutils.TruncateTables(s.db)
}

func (s *ServerTestSuite) TearDownSuite() {
	s.ctxCancel()

	utils.Close(s.sc, s.logger)
	utils.Close(s.db, s.logger)
}

func TestRunServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

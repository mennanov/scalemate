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

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/credentials"

	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"
)

const (
	dbName = "scheduler_server_test_suite"
)

type ServerTestSuite struct {
	suite.Suite
	service         *server.SchedulerServer
	client          scheduler_proto.SchedulerClient
	db              *sqlx.DB
	sc              stan.Conn
	subscription    events.Subscription
	messagesHandler *testutils.MessagesTestingHandler
	ctxCancel       context.CancelFunc
	logger          *logrus.Logger
	claimsInjector  *auth.FakeClaimsInjector
}

func (s *ServerTestSuite) SetupSuite() {
	logger := logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(logger)

	// Start gRPC server.
	localAddr := "localhost:50052"

	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	db, err := testutils.CreateTestingDatabase(conf.SchdulerConf.DBUrl, dbName)
	s.Require().NoError(err)
	s.db = db

	s.claimsInjector = auth.NewFakeClaimsContextInjector(&auth.Claims{
		Username:  "test_username",
		TokenType: auth.TokenTypeAccess,
	})

	s.sc, err = stan.Connect(
		conf.SchdulerConf.NatsClusterName,
		fmt.Sprintf("scheduler-service-%s", uuid.New().String()),
		stan.NatsURL(conf.SchdulerConf.NatsAddr))
	s.Require().NoError(err)

	producer := events.NewNatsProducer(s.sc, events.SchedulerSubjectName, 5)
	consumer := events.NewNatsConsumer(s.sc, events.SchedulerSubjectName, producer, s.db, s.logger, 5, 3)
	s.messagesHandler = testutils.NewMessagesTestingHandler()
	s.subscription, err = consumer.Consume(s.messagesHandler)

	s.service = server.NewSchedulerServer(
		server.WithLogger(logger),
		server.WithDBConnection(s.db),
		server.WithProducer(producer),
		server.WithClaimsInjector(s.claimsInjector),
	)

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

	serverCreds, err := credentials.NewServerTLSFromFile(conf.SchdulerConf.TLSCertFile, conf.SchdulerConf.TLSKeyFile)
	s.Require().NoError(err)

	go func() {
		s.service.Serve(ctx, localAddr, serverCreds)
	}()

	clientCreds, err := credentials.NewClientTLSFromFile(conf.SchdulerConf.TLSCertFile, "localhost")
	s.Require().NoError(err)
	s.client = scheduler_proto.NewSchedulerClient(testutils.NewClientConn(localAddr, clientCreds))
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

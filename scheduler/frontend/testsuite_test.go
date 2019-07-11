package frontend_test

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

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/credentials"

	"github.com/mennanov/scalemate/scheduler/backend"
	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/scheduler/frontend"
	"github.com/mennanov/scalemate/scheduler/handlers"
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
	frontEndService                 *frontend.SchedulerFrontend
	frontEndClient                  scheduler_proto.SchedulerFrontEndClient
	backEndService                  *backend.SchedulerBackend
	backEndClient                   scheduler_proto.SchedulerBackEndClient
	db                              *sqlx.DB
	sc                              stan.Conn
	producer                        events.Producer
	messagesHandler                 *testutils.ExpectMessagesHandler
	ctxCancel                       context.CancelFunc
	logger                          *logrus.Logger
	claimsInjector                  *auth.FakeClaimsInjector
}

func (s *ServerTestSuite) SetupSuite() {
	logger := logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(logger)

	// Start gRPC server.
	frontEndAddr := "localhost:50052"
	backEndAddr := "localhost:50053"

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

	s.producer = events.NewNatsProducer(s.sc, events.SchedulerSubjectName, 5)

	s.frontEndService = frontend.NewSchedulerServer(
		frontend.WithLogger(logger),
		frontend.WithDBConnection(s.db),
		frontend.WithProducer(s.producer),
		frontend.WithClaimsInjector(s.claimsInjector),
	)
	s.backEndService = backend.NewSchedulerBackend(
		backend.WithLogger(logger),
		backend.WithDBConnection(s.db),
		backend.WithProducer(s.producer),
		backend.WithClaimsInjector(s.claimsInjector),
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

	go s.frontEndService.Serve(ctx, frontEndAddr, serverCreds)
	go s.backEndService.Serve(ctx, backEndAddr, serverCreds)
	time.Sleep(time.Millisecond * 500)

	clientCreds, err := credentials.NewClientTLSFromFile(conf.SchdulerConf.TLSCertFile, "localhost")
	s.Require().NoError(err)
	s.frontEndClient = scheduler_proto.NewSchedulerFrontEndClient(testutils.NewClientConn(frontEndAddr, clientCreds))
	s.backEndClient = scheduler_proto.NewSchedulerBackEndClient(testutils.NewClientConn(backEndAddr, clientCreds))
}

func (s *ServerTestSuite) SetupTest() {
	testutils.TruncateTables(s.db)
}

// runEventHandlers runs event handlers and returns a function that closes the subscription when called.
func (s *ServerTestSuite) runEventHandlers() func() {
	sub, err := s.sc.Subscribe(
		events.SchedulerSubjectName,
		events.StanMsgHandler(context.Background(), s.logger, 1, handlers.NewIndependentHandlers(s.db, s.producer, s.logger, 5)...),
		stan.SetManualAckMode())
	s.Require().NoError(err)
	return func() {
		s.Require().NoError(sub.Close())
	}
}

func (s *ServerTestSuite) TearDownSuite() {
	s.ctxCancel()

	utils.Close(s.sc, s.logger)
	utils.Close(s.db, s.logger)
}

func TestRunServerSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

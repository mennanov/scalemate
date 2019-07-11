package handlers_test

import (
	"fmt"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // keep
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"
)

const (
	dbName = "scheduler_handlers_test_suite"
)

type HandlersTestSuite struct {
	suite.Suite
	db       *sqlx.DB
	producer *events.FakeProducer
	logger   *logrus.Logger
	conn     stan.Conn
}

func (s *HandlersTestSuite) SetupSuite() {
	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	db, err := testutils.CreateTestingDatabase(conf.SchdulerConf.DBUrl, dbName)
	s.Require().NoError(err)
	s.db = db
	s.producer = events.NewFakeProducer()

	s.conn, err = stan.Connect(
		conf.SchdulerConf.NatsClusterName,
		// Unique clientID is used to avoid interference with events from the previous test runs.
		fmt.Sprintf("scheduler-events-handler-%s", uuid.New().String()),
		stan.NatsURL(conf.SchdulerConf.NatsAddr))

	// Run migrations.
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{
		MigrationsTable: "migrations",
		DatabaseName:    dbName,
		SchemaName:      "",
	})
	m, err := migrate.NewWithDatabaseInstance("file://../migrations", dbName, driver)
	s.Require().NoError(err)
	s.Require().NoError(m.Up())
}

func (s *HandlersTestSuite) SetupTest() {
	testutils.TruncateTables(s.db)
	s.producer.SentEvents = nil
}

func (s *HandlersTestSuite) TearDownSuite() {
	utils.Close(s.conn, s.logger)
	utils.Close(s.db, s.logger)
}

func TestRunHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

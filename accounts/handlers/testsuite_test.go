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

	"github.com/mennanov/scalemate/accounts/conf"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"
)

const (
	natsDurableName = "accounts-handlers-tests"
	dbName          = "accounts_handlers_test_suite"
)

type HandlersTestSuite struct {
	suite.Suite
	db              *sqlx.DB
	logger          *logrus.Logger
	conn            stan.Conn
	subscription    events.Subscription
	messagesHandler *events.MessagesTestingHandler
}

func (s *HandlersTestSuite) SetupSuite() {
	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	db, err := testutils.CreateTestingDatabase(conf.AccountsConf.DBUrl, dbName)
	s.Require().NoError(err)
	s.db = db

	s.conn, err = stan.Connect(
		conf.AccountsConf.NatsClusterName,
		fmt.Sprintf("accounts-events-handler-%s", uuid.New().String()),
		stan.NatsURL(conf.AccountsConf.NatsAddr))

	consumer := events.NewNatsConsumer(s.conn, events.AccountsSubjectName, s.logger, stan.DurableName(natsDurableName))
	s.messagesHandler = events.NewMessagesTestingHandler()
	s.subscription, err = consumer.Consume(s.messagesHandler)
	s.Require().NoError(err)

	// Run migrations.
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{
		MigrationsTable: "migrations",
		DatabaseName:    "accounts_models_test_suite",
		SchemaName:      "",
	})
	m, err := migrate.NewWithDatabaseInstance("file://../migrations", dbName, driver)
	s.Require().NoError(err)
	s.Require().NoError(m.Up())
}

func (s *HandlersTestSuite) SetupTest() {
	testutils.TruncateTables(s.db)
}

func (s *HandlersTestSuite) TearDownSuite() {
	utils.Close(s.subscription, s.logger)
	utils.Close(s.conn, s.logger)
	utils.Close(s.db, s.logger)
}

func TestRunHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

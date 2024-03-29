package handlers_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"
)

const natsDurableName = "scheduler-handlers-tests"

type HandlersTestSuite struct {
	suite.Suite
	db              *gorm.DB
	logger          *logrus.Logger
	conn            stan.Conn
	subscription    events.Subscription
	messagesHandler *events.MessagesTestingHandler
}

func (s *HandlersTestSuite) SetupSuite() {
	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	db, err := testutils.CreateTestingDatabase(conf.SchdulerConf.DBUrl, "scheduler_handlers_test_suite")
	s.Require().NoError(err)
	s.db = db.LogMode(s.logger.IsLevelEnabled(logrus.DebugLevel))

	s.conn, err = stan.Connect(
		conf.SchdulerConf.NatsClusterName,
		fmt.Sprintf("scheduler-events-handler-%s", uuid.New().String()),
		stan.NatsURL(conf.SchdulerConf.NatsAddr))

	consumer := events.NewNatsConsumer(s.conn, events.SchedulerSubjectName, s.logger, stan.DurableName(natsDurableName))
	s.messagesHandler = &events.MessagesTestingHandler{}
	s.subscription, err = consumer.Consume(s.messagesHandler)
	s.Require().NoError(err)

	s.Require().NoError(migrations.RunMigrations(s.db))
}

func (s *HandlersTestSuite) SetupTest() {
	testutils.TruncateTables(s.db, s.logger, models.TableNames...)
}

func (s *HandlersTestSuite) TearDownSuite() {
	utils.Close(s.subscription, s.logger)
	utils.Close(s.conn, s.logger)
	utils.Close(s.db, s.logger)
}

func TestRunHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

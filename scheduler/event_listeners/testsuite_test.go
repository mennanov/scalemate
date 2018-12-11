package event_listeners_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/suite"
	"gopkg.in/khaiql/dbcleaner.v2"

	"gopkg.in/khaiql/dbcleaner.v2/engine"

	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

var cleaner = dbcleaner.New()

// EventListenersTestSuite is a test suite for event listeners.
type EventListenersTestSuite struct {
	suite.Suite
	service *server.SchedulerServer
}

func (s *EventListenersTestSuite) SetupSuite() {
	var err error
	db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
	s.Require().NoError(err)

	s.service = &server.SchedulerServer{
		DB:               db,
		NewTasksByNodeID: make(map[uint64]chan *scheduler_proto.Task),
		NewTasksByJobID:  make(map[uint64]chan *scheduler_proto.Task),
	}

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.service.DB.LogMode(true)))

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv(server.DBEnvConf.Host), os.Getenv(server.DBEnvConf.Port), os.Getenv(server.DBEnvConf.User),
		os.Getenv(server.DBEnvConf.Name), os.Getenv(server.DBEnvConf.Password))
	pg := engine.NewPostgresEngine(dsn)
	cleaner.SetEngine(pg)
}

func (s *EventListenersTestSuite) SetupTest() {
	s.service.Publisher = events.NewFakePublisher()

	for _, tableName := range models.TableNames {
		cleaner.Acquire(tableName)
	}
}

func (s *EventListenersTestSuite) TearDownTest() {
	for _, tableName := range models.TableNames {
		cleaner.Clean(tableName)
	}
}

// TODO: uncomment this when the deadlock problem is resolved.
//func (s *ModelsTestSuite) TearDownSuite() {
//	migrations.RollbackAllMigrations(s.db)
//}

func TestRunEventListenersSuite(t *testing.T) {
	suite.Run(t, new(EventListenersTestSuite))
}

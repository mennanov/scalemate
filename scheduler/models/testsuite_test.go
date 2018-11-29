package models_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"
	"gopkg.in/khaiql/dbcleaner.v2"
	"gopkg.in/khaiql/dbcleaner.v2/engine"

	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/server"
)

var cleaner = dbcleaner.New()

type ModelsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *ModelsTestSuite) SetupSuite() {
	srv, err := server.NewSchedulerServerFromEnv(server.SchedulerEnvConf)
	s.Require().NoError(err)
	s.db = srv.DB

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.db.LogMode(true)))

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv(server.DBEnvConf.Host), os.Getenv(server.DBEnvConf.Port), os.Getenv(server.DBEnvConf.User),
		os.Getenv(server.DBEnvConf.Name), os.Getenv(server.DBEnvConf.Password))
	pg := engine.NewPostgresEngine(dsn)
	cleaner.SetEngine(pg)
}

func (s *ModelsTestSuite) SetupTest() {
	cleaner.Acquire("nodes")
	cleaner.Acquire("tasks")
	cleaner.Acquire("jobs")
}

func (s *ModelsTestSuite) TearDownTest() {
	cleaner.Clean("nodes")
	cleaner.Clean("tasks")
	cleaner.Clean("jobs")
}

// TODO: uncomment this when the deadlock problem is resolved.
//func (s *ModelsTestSuite) TearDownSuite() {
//	migrations.RollbackAllMigrations(s.db)
//}

func TestRunModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

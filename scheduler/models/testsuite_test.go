package models_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"
	"gopkg.in/khaiql/dbcleaner.v2"
	"gopkg.in/khaiql/dbcleaner.v2/engine"
	"gopkg.in/khaiql/dbcleaner.v2/logging"

	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/utils"
)

var cleaner = dbcleaner.New(dbcleaner.SetLogger(&logging.Stdout{}))

type ModelsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *ModelsTestSuite) SetupSuite() {
	db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
	s.Require().NoError(err)
	s.db = db

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.db.LogMode(true)))

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv(server.DBEnvConf.Host), os.Getenv(server.DBEnvConf.Port), os.Getenv(server.DBEnvConf.User),
		os.Getenv(server.DBEnvConf.Name), os.Getenv(server.DBEnvConf.Password))
	pg := engine.NewPostgresEngine(dsn)
	cleaner.SetEngine(pg)
}

func (s *ModelsTestSuite) SetupTest() {
	for _, tableName := range models.TableNames {
		cleaner.Acquire(tableName)
	}
}

func (s *ModelsTestSuite) TearDownTest() {
	for _, tableName := range models.TableNames {
		cleaner.Clean(tableName)
	}
}

func TestRunModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

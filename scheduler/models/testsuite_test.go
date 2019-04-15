package models_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

type ModelsTestSuite struct {
	suite.Suite
	db     *gorm.DB
	logger *logrus.Logger
}

func (s *ModelsTestSuite) SetupSuite() {
	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	db, err := utils.CreateTestingDatabase(conf.SchdulerConf.DBUrl, "scheduler_models_test_suite")
	s.Require().NoError(err)
	s.db = db.LogMode(s.logger.IsLevelEnabled(logrus.DebugLevel))

	// Prepare database.
	s.Require().NoError(migrations.RunMigrations(s.db))
}

func (s *ModelsTestSuite) SetupTest() {
	utils.TruncateTables(s.db, s.logger, models.TableNames...)
}

func TestRunModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

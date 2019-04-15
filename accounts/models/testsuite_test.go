package models_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/accounts/conf"
	"github.com/mennanov/scalemate/accounts/migrations"
	"github.com/mennanov/scalemate/accounts/models"
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

	db, err := utils.CreateTestingDatabase(conf.AccountsConf.DBUrl, "accounts_models_test_suite")
	s.Require().NoError(err)
	s.db = db.LogMode(s.logger.IsLevelEnabled(logrus.DebugLevel))

	s.Require().NoError(migrations.RunMigrations(s.db))
}

func (s *ModelsTestSuite) TearDownTest() {
	utils.TruncateTables(s.db, s.logger, models.TableNames...)
}

func (s *ModelsTestSuite) TearDownSuite() {
	s.NoError(migrations.RollbackAllMigrations(s.db))
	utils.Close(s.db, s.logger)
}

func TestRunModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

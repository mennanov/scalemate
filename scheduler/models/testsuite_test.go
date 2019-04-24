package models_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"
)

type ModelsTestSuite struct {
	suite.Suite
	db *gorm.DB
	logger *logrus.Logger
}

func (s *ModelsTestSuite) SetupSuite() {
	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	db, err := testutils.CreateTestingDatabase(conf.SchdulerConf.DBUrl, "scheduler_models_test_suite")
	s.Require().NoError(err)
	sqlx.Open()
	// Prepare database.
}

func (s *ModelsTestSuite) SetupTest() {
	testutils.TruncateTables(s.gormDB, s.logger, models.TableNames...)
}

func TestRunModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

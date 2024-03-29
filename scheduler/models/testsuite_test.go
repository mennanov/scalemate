package models_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // keep
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"
)

const dbName = "scheduler_models_test_suite"

type ModelsTestSuite struct {
	suite.Suite
	db     *sqlx.DB
	logger *logrus.Logger
}

func (s *ModelsTestSuite) SetupSuite() {
	s.logger = logrus.StandardLogger()
	utils.SetLogrusLevelFromEnv(s.logger)

	db, err := testutils.CreateTestingDatabase(conf.SchdulerConf.DBUrl, dbName)
	s.Require().NoError(err)
	s.db = db

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

func (s *ModelsTestSuite) SetupTest() {
	testutils.TruncateTables(s.db)
}

func (s *ModelsTestSuite) TearDownSuite() {
	utils.Close(s.db, s.logger)
}

func TestRunModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

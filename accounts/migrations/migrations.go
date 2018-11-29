package migrations

import (
	"github.com/jinzhu/gorm"
	"github.com/mennanov/gormigrate"

	"github.com/mennanov/scalemate/accounts/models"
)

var options = &gormigrate.Options{
	TableName:      "migrations",
	IDColumnName:   "id",
	IDColumnSize:   255,
	UseTransaction: true,
}

var migrations = []*gormigrate.Migration{
	{
		ID: "20180722_create_users",
		Migrate: func(tx *gorm.DB) error {
			// TODO: inline the model's struct here.
			return tx.AutoMigrate(&models.User{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTable("users").Error
		},
	},
	{
		ID: "20181031_create_node",
		Migrate: func(tx *gorm.DB) error {
			// TODO: inline the model's struct here.
			return tx.AutoMigrate(&models.Node{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTable("nodes").Error
		},
	},
}

// RunMigrations runs all the un-applied migrations.
func RunMigrations(db *gorm.DB) error {
	m := gormigrate.New(db, options, migrations)
	return m.Migrate()
}

// RollbackMigrations rolls back migrations to a certain migration by its ID.
func RollbackMigrations(db *gorm.DB, migrationID string) error {
	m := gormigrate.New(db, options, migrations)
	if migrationID != "" {
		return m.RollbackTo(migrationID)
	}
	return m.RollbackLast()
}

// RollbackAllMigrations rolls back all migrations.
func RollbackAllMigrations(db *gorm.DB) error {
	m := gormigrate.New(db, options, migrations)
	return m.RollbackTo(migrations[0].ID)
}

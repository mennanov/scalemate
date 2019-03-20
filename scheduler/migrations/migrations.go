package migrations

import (
	"github.com/jinzhu/gorm"
	"github.com/go-gormigrate/gormigrate"

	"github.com/mennanov/scalemate/scheduler/models"
)

var options = &gormigrate.Options{
	TableName:      "migrations",
	IDColumnName:   "id",
	IDColumnSize:   255,
	UseTransaction: true,
}

var migrations = []*gormigrate.Migration{
	{
		ID: "20180827_create_job",
		Migrate: func(tx *gorm.DB) error {
			// TODO: inline the model struct here once we have real migrations later.
			return tx.AutoMigrate(&models.Job{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTable("jobs").Error
		},
	},
	{
		ID: "20180827_create_node",
		Migrate: func(tx *gorm.DB) error {
			// TODO: inline the model struct here once we have real migrations later.
			return tx.AutoMigrate(&models.Node{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTable("nodes").Error
		},
	},
	{
		ID: "20180827_create_node_index",
		Migrate: func(tx *gorm.DB) error {
			return tx.Model(&models.Node{}).AddIndex(
				"idx_node_schedule_index",
				"deleted_at",
				"status",
				"cpu_class_min",
				"cpu_class",
				"cpu_model",
				"disk_class_min",
				"disk_class",
				"disk_model",
				"gpu_class_min",
				"gpu_class",
				"gpu_model",
				"memory_model",
				"username",
				"labels",
				"cpu_available",
				"memory_available",
				"disk_available",
				"gpu_available",
			).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.RemoveIndex("idx_node_schedule_index").Error
		},
	},
	{
		ID: "20180828_create_task",
		Migrate: func(tx *gorm.DB) error {
			// TODO: inline the model struct here once we have real migrations later.
			return tx.AutoMigrate(&models.Task{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTable("tasks").Error
		},
	},
	{
		ID: "20180921_create_job_index",
		Migrate: func(tx *gorm.DB) error {
			return tx.Model(&models.Job{}).AddIndex(
				"idx_job_schedule_index",
				"deleted_at",
				"status",
				"cpu_class",
				"gpu_class",
				"disk_class",
				"cpu_limit",
				"memory_limit",
				"disk_limit",
				"gpu_limit",
			).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.RemoveIndex("idx_job_schedule_index").Error
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

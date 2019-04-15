package models

import (
	"github.com/jinzhu/gorm"
)

// TableNames is a list of all the registered table names that is used in tests to clean up database.
var TableNames = []string{"tasks", "jobs", "nodes", "processed_events"}

func init() {
	// Remove default gorm callbacks on Create as it populates the `UpdatedAt` field which is undesirable.
	gorm.DefaultCallback.Create().Remove("gorm:update_time_stamp")
	gorm.DefaultCallback.Create().Remove("gorm:update_time_stamp_when_create")
}

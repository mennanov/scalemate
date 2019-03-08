package models

import (
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
)

// TableNames is a list of all the registered table names that is used in tests to clean up database.
var TableNames = [3]string{"tasks", "jobs", "nodes"}

// Model struct is a copy of gorm.Model, but with the ID field of uint64 instead of unit.
type Model struct {
	ID        uint64     `gorm:"primary_key"`
	CreatedAt time.Time  `gorm:"not null;"`
	UpdatedAt *time.Time `gorm:"default:null"`
	DeletedAt *time.Time `sql:"index"`
}

func init() {
	// Remove default gorm callbacks on Create as it populates the `UpdatedAt` field which is undesirable.
	gorm.DefaultCallback.Create().Remove("gorm:update_time_stamp")
	gorm.DefaultCallback.Create().Remove("gorm:update_time_stamp_when_create")
}

// BeforeCreate is a gorm callback that is called before every Create operation. It populates the `CreatedAt` field with
// the current time.
func (m *Model) BeforeCreate(scope *gorm.Scope) error {
	if m.CreatedAt.IsZero() {
		return scope.SetColumn("CreatedAt", time.Now())
	}
	return nil
}

// Enum represents protobuf ENUM fields.
// This is needed to provide a custom implementation for the String() method which is used by the ORM in SQL queries.
type Enum int32

// String returns a string representation of the Enum (int32), e.g. "5" for 5.
func (s *Enum) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

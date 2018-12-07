package models

import (
	"strconv"
	"time"
)

// TableNames is a list of all the registered table names that is used in tests to clean up database.
var TableNames = [3]string{"nodes", "jobs", "tasks"}

// Model struct is a copy of gorm.Model, but with the ID field of uint64 instead of unit.
type Model struct {
	ID        uint64 `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// Enum represents protobuf ENUM fields.
// This is needed to provide a custom implementation for the String() method which is used by the ORM in SQL queries.
type Enum int32

// String returns a string representation of the Enum (int32), e.g. "5" for 5.
func (s *Enum) String() string {
	return strconv.FormatInt(int64(*s), 10)
}

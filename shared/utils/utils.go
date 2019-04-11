// Package utils provides convenience functions to be used in services (especially in tests).
package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // keep
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Close is a wrapper around io.Closer that logs the error returned by the io.Closer.
func Close(c io.Closer, logger *logrus.Logger) {
	if err := c.Close(); err != nil {
		logger.WithError(err).Errorf("%+v", errors.Wrap(err, "failed to Close a resource"))
	}
}

// ConnectDBFromEnv creates a database connection from environment variables.
// All the parameters may be overwritten.
func ConnectDBFromEnv(url string) (*gorm.DB, error) {
	// TODO: enable SSL mode for production.
	var i time.Duration
	for i = 1; i <= 16; i *= 2 {
		db, err := gorm.Open("postgres", url)

		if err != nil {
			logrus.WithField("url", url).Errorf("Could not connect to DB: %s", err)

			d := i * time.Second
			logrus.Infof("Retrying to connect to DB in %s sec", d)
			time.Sleep(d)
			continue
		}
		return db, nil
	}

	return nil, errors.New("Could not connect to DB")
}

// HandleDBError creates a gRPC status error from the failed DB query.
func HandleDBError(db *gorm.DB) error {
	if db.Error == nil {
		return nil
	}
	if db.RecordNotFound() {
		return errors.WithStack(status.Error(codes.NotFound, db.Error.Error()))
	}
	errString := db.Error.Error()
	if strings.Contains(errString, "duplicate") {
		return errors.WithStack(status.Error(codes.AlreadyExists, db.Error.Error()))
	}
	return errors.WithStack(status.Error(codes.Internal, db.Error.Error()))
}

// RollbackTransaction rolls back the given transaction. It returns the given precedingError if the rollback is
// successful, otherwise returns a wrapped precedingError.
func RollbackTransaction(tx *gorm.DB, precedingError error) error {
	if txErr := HandleDBError(tx.Rollback()); txErr != nil {
		return errors.Wrapf(precedingError, "failed to rollback transaction: %s", txErr.Error())
	}
	return precedingError
}

// SetLogrusLevelFromEnv sets the log level for logrus from environment variables.
func SetLogrusLevelFromEnv(logger *logrus.Logger) {
	lvl := os.Getenv("LOG_LEVEL")
	switch lvl {
	case "DEBUG":
		logger.SetLevel(logrus.DebugLevel)
	case "INFO":
		logger.SetLevel(logrus.InfoLevel)
	case "WARN":
		logger.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
}

// Model struct is a copy of gorm.Model, but with the ID field of uint64 instead of unit.
type Model struct {
	ID        uint64     `gorm:"primary_key"`
	CreatedAt time.Time  `gorm:"not null;"`
	UpdatedAt *time.Time `gorm:"default:null"`
	DeletedAt *time.Time `sql:"index"`
}

// BeforeCreate is a gorm callback that is called before every Create operation. It populates the `CreatedAt` field with
// the current time.
func (m *Model) BeforeCreate(scope *gorm.Scope) error {
	if m.CreatedAt.IsZero() {
		return scope.SetColumn("CreatedAt", time.Now().UTC())
	}
	return nil
}

// Enum represents protobuf ENUM fields.
// This is needed to provide a custom implementation for the String() method which is used by the ORM in SQL queries.
type Enum int32

// String returns a string representation of the Enum (int32), e.g. "5" for 5.
func (s *Enum) String() string {
	return fmt.Sprintf("%d", s)
}

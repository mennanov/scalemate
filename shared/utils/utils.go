// Package utils provides convenience functions to be used in services (especially in tests).
package utils

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // keep
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/events"
)

// Close is a wrapper around io.Closer that logs the error returned by the io.Closer.
func Close(c io.Closer, logger *logrus.Logger) {
	if err := c.Close(); err != nil {
		logger.WithError(err).Errorf("%+v", errors.Wrap(err, "failed to Close a resource"))
	}
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

// ConnectDBFromEnv creates a database connection from environment variables.
// All the parameters may be overwritten.
func ConnectDBFromEnv(url string) (*sqlx.DB, error) {
	// TODO: enable SSL mode for production.
	var i time.Duration
	for i = 1; i <= 16; i *= 2 {
		db, err := sqlx.Open("postgres", url)

		if err != nil {
			logrus.WithField("url", url).Errorf("Could not connect to DB: %s", err)

			d := i * time.Second
			logrus.Infof("Retrying to connect to DB in %s sec", d)
			time.Sleep(d)
			continue
		}
		db.MapperFunc(strcase.ToSnake)
		return db, nil
	}

	return nil, errors.New("Could not connect to DB")
}

// HandleDBError creates a gRPC status error from the failed DB query.
func HandleDBError(err error) error {
	if err == nil {
		return nil
	}
	errString := err.Error()
	if err == sql.ErrNoRows {
		return errors.WithStack(status.Error(codes.NotFound, errString))
	}

	if strings.Contains(errString, "duplicate") {
		return errors.WithStack(status.Error(codes.AlreadyExists, errString))
	}
	return errors.WithStack(status.Error(codes.Internal, errString))
}

// RollbackTransaction rolls back the given transaction. It returns the given precedingError if the rollback is
// successful, otherwise returns a wrapped precedingError.
func RollbackTransaction(tx driver.Tx, precedingError error) error {
	if txErr := HandleDBError(tx.Rollback()); txErr != nil {
		return errors.Wrapf(precedingError, "failed to rollback transaction: %s", txErr.Error())
	}
	return precedingError
}

// Model struct is a copy of gorm.Model, but with the ID field of uint64 instead of unit.
type Model struct {
	ID        uint32
	CreatedAt time.Time
	UpdatedAt *time.Time
}

// Enum represents protobuf ENUM fields.
type Enum int32

// String returns a string representation of the Enum (int32), e.g. "5" for 5.
func (s Enum) String() string {
	return fmt.Sprintf("%d", s)
}

// CommitAndPublish commits the DB transaction and sends the given events with confirmation.
// It also handles and wraps all the errors if there is any.
func CommitAndPublish(tx driver.Tx, producer events.Producer, events ...*events_proto.Event) error {
	if len(events) > 0 {
		if err := producer.Send(events...); err != nil {
			// Failed to confirm sent events: rollback the transaction.
			return RollbackTransaction(tx, errors.Wrap(err, "producer.Send failed"))
		}
	}
	// TODO: messages sent to NATS may be delivered to consumers earlier than they are committed to DB by the
	//  corresponding producers. Need to add a delay in all consumers that query the same database as the corresponding
	//  producers.
	if err := HandleDBError(tx.Commit()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

// SqlxGetter is a missing interface in sqlx library for the Get method.
type SqlxGetter interface {
	Get(dest interface{}, query string, args ...interface{}) error
}

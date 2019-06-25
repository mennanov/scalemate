// Package utils provides convenience functions to be used in services (especially in tests).
package utils

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"

	"github.com/iancoleman/strcase"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // keep
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
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
	if strings.Contains(errString, "serialize") {
		// Serializable transaction has failed. Is should be retried by the client.
		// Expected error message from postgres: "ERROR:  could not serialize access due to concurrent update".
		return errors.WithStack(status.Error(codes.Aborted, errString))
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

// Enum represents protobuf ENUM fields.
type Enum int32

// String returns a string representation of the Enum (int32), e.g. "5" for 5.
func (s Enum) String() string {
	return fmt.Sprintf("%d", s)
}

// SqlxGetter is a missing interface in sqlx library for the Get method.
type SqlxGetter interface {
	Get(dest interface{}, query string, args ...interface{}) error
}

// SqlxSelector is a missing interface in sqlx library for the Select method.
type SqlxSelector interface {
	Select(dest interface{}, query string, args ...interface{}) error
}

// SqlxNamedQuery is a missing interface in sqlx library for the NamedQuery method.
type SqlxNamedQuery interface {
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
}

// SqlxExtGetter is a union interface of sqlx.Ext, SqlxGetter and SqlxSelector.
type SqlxExtGetter interface {
	sqlx.Ext
	SqlxGetter
	SqlxSelector
}

// ClaimsUsernameEqual checks if the claims' Username value equals to the given username string.
func ClaimsUsernameEqual(ctx context.Context, username string) error {
	ctxClaims := ctx.Value(auth.ContextKeyClaims)
	if ctxClaims == nil {
		return status.Error(codes.Unauthenticated, "no JWT claims found")
	}
	claims, ok := ctxClaims.(*auth.Claims)
	if !ok {
		return status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	if claims.Username != username {
		return status.Error(codes.PermissionDenied, "permission denied")
	}
	return nil
}

type isEmpty struct {
}

// ValidateIsEmpty checks if the value is empty.
var ValidateIsEmpty = new(isEmpty)

// Validate checks whether the given value is empty.
func (i *isEmpty) Validate(value interface{}) error {
	if !validation.IsEmpty(value) {
		return status.Errorf(codes.InvalidArgument, "readonly field is not empty", value)
	}
	return nil
}

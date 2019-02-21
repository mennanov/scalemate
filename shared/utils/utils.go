// Package utils provides convenience functions to be used in services (especially in tests).
package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// Close is a wrapper around io.Closer that logs the error returned by the io.Closer.
func Close(c io.Closer) {
	if err := c.Close(); err != nil {
		logrus.Errorf("%+v", errors.Wrap(err, "failed to Close a resource"))
	}
}

// DBEnvConf maps DB connection settings to environment variable names.
type DBEnvConf struct {
	Host     string
	Port     string
	User     string
	Name     string
	Password string
}

// ConnectDBFromEnv creates a database connection from environment variables.
// All the parameters may be overwritten.
func ConnectDBFromEnv(conf DBEnvConf) (*gorm.DB, error) {
	host := os.Getenv(conf.Host)
	port := os.Getenv(conf.Port)
	user := os.Getenv(conf.User)
	name := os.Getenv(conf.Name)
	password := os.Getenv(conf.Password)

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", host, port, user, name, password)

	// TODO: enable SSL mode for production.
	var i time.Duration
	for i = 1; i <= 16; i *= 2 {
		db, err := gorm.Open("postgres", connectionString)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"DBEnvConf":        conf,
				"connectionString": connectionString,
			}).Errorf("Could not connect to DB: %s", err)

			d := i * time.Second
			logrus.Infof("Retrying to connect to DB in %s sec", d)
			time.Sleep(d)
			continue
		}
		return db, nil
	}

	return nil, errors.New("Could not connect to DB")
}

// AMQPEnvConf maps AMQP connection settings to environment variable names.
type AMQPEnvConf struct {
	Addr string
}

// TLSEnvConf maps TLS files settings to environment variable names.
type TLSEnvConf struct {
	CertFile string
	KeyFile  string
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

// TLSServerCredentialsFromEnv creates gRPC server TLS credentials from environment variables.
func TLSServerCredentialsFromEnv(conf TLSEnvConf) credentials.TransportCredentials {
	certFile := os.Getenv(conf.CertFile)

	if certFile == "" {
		panic(fmt.Sprintf("%s env variable is empty", conf.CertFile))
	}

	keyFile := os.Getenv(conf.KeyFile)

	if keyFile == "" {
		panic(fmt.Sprintf("%s env variable is empty", conf.KeyFile))
	}

	// Bazel specific path to data files. See https://docs.bazel.build/versions/master/build-ref.html#data
	srcDir := os.Getenv("TEST_SRCDIR")
	if srcDir != "" {
		certFile = path.Join(srcDir, certFile)
		keyFile = path.Join(srcDir, keyFile)
	}

	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		panic(err)
	}

	return creds
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

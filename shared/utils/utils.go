// Package utils provides convenience functions to be used in services (especially in tests).
package utils

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/protobuf/field_mask"
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

// SendAndCommit sends the given events with confirmation and commits or rolls back the DB transaction.
// It also handles and wraps all the errors if there is any.
func SendAndCommit(tx *gorm.DB, publisher Publisher, events ...*events_proto.Event) error {
	if err := publisher.Send(events...); err != nil {
		// Failed to confirm sent events: rollback the transaction.
		if txErr := HandleDBError(tx.Rollback()); txErr != nil {
			return errors.Wrapf(txErr, "failed to rollback transaction in SendAndCommit: %s", err.Error())
		}
		return errors.Wrap(err, "failed to send events with confirmation")
	}
	if err := HandleDBError(tx.Commit()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
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
func SetLogrusLevelFromEnv() {
	lvl := os.Getenv("LOG_LEVEL")
	switch lvl {
	case "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}

// RoutingKeyFromEvent generates an AMQP routing key for a topic exchange to be used to publish messages.
// The key is in form of "entity_type.event_type[.field.mask.fields]", e.g. "node.updated.status.connected_at".
func RoutingKeyFromEvent(event *events_proto.Event) (string, error) {
	var entityName string
	switch event.Payload.(type) {
	case *events_proto.Event_SchedulerNode:
		entityName = "scheduler.node"

	case *events_proto.Event_SchedulerJob:
		entityName = "scheduler.job"

	case *events_proto.Event_SchedulerTask:
		entityName = "scheduler.task"

	case *events_proto.Event_AccountsUser:
		entityName = "accounts.user"

	case *events_proto.Event_AccountsNode:
		entityName = "accounts.node"

	default:
		return "", errors.Errorf("unexpected type %T", event.Payload)
	}

	var eventType string
	switch event.Type {
	case events_proto.Event_CREATED:
		eventType = "created"

	case events_proto.Event_UPDATED:
		eventType = "updated"

	case events_proto.Event_DELETED:
		eventType = "deleted"

	default:
		return "", errors.Errorf("unexpected Event.Type %T", event.Type)
	}

	var fieldMask string
	if event.PayloadMask != nil && len(event.PayloadMask.Paths) > 0 {
		fieldMask = fmt.Sprintf(".%s", strings.Join(event.PayloadMask.Paths, "."))
	}

	// TODO: max AMQP routing key length is 255. Need to handle cases when it's not enough (e.g. fieldMask is too big).
	return fmt.Sprintf("%s.%s%s", entityName, eventType, fieldMask), nil
}

// NewEventFromPayload creates a new events_proto.Event instance for the given model, event type and service.
func NewEventFromPayload(payload proto.Message, eventType events_proto.Event_Type, service events_proto.Service, fieldMask *field_mask.FieldMask) (*events_proto.Event, error) {
	createdAt, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
	}

	event := &events_proto.Event{
		Type:        eventType,
		Service:     service,
		PayloadMask: fieldMask,
		CreatedAt:   createdAt,
	}

	switch p := payload.(type) {
	case *scheduler_proto.Job:
		event.Payload = &events_proto.Event_SchedulerJob{SchedulerJob: p}

	case *scheduler_proto.Node:
		event.Payload = &events_proto.Event_SchedulerNode{SchedulerNode: p}

	case *scheduler_proto.Task:
		event.Payload = &events_proto.Event_SchedulerTask{SchedulerTask: p}

	case *accounts_proto.User:
		event.Payload = &events_proto.Event_AccountsUser{AccountsUser: p}

	case *accounts_proto.Node:
		event.Payload = &events_proto.Event_AccountsNode{AccountsNode: p}

	default:
		return nil, errors.Errorf("unexpected payload type %T", payload)
	}

	return event, nil
}

// NewModelProtoFromEvent creates a corresponding proto representation of the event's payload.
func NewModelProtoFromEvent(event *events_proto.Event) (proto.Message, error) {
	var msg proto.Message
	switch p := event.Payload.(type) {
	case *events_proto.Event_AccountsUser:
		msg = p.AccountsUser

	case *events_proto.Event_SchedulerNode:
		msg = p.SchedulerNode

	case *events_proto.Event_SchedulerJob:
		msg = p.SchedulerJob

	case *events_proto.Event_SchedulerTask:
		msg = p.SchedulerTask

	default:
		return nil, errors.Errorf("unexpected event payload type %T", event.Payload)
	}

	return msg, nil
}



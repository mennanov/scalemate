package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// SchedulerAMQPExchangeName is the name of the exchange to be used to send all the events from this service.
	SchedulerAMQPExchangeName = "scheduler_events"
	// AccountsAMQPExchangeName is an AMQP exchange name for all the Accounts service events.
	AccountsAMQPExchangeName = "accounts_events"
)

// AMQPExchangeDeclare declares an AMQP exchange.
func AMQPExchangeDeclare(ch *amqp.Channel, exchangeName string) error {
	return ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

// CommitAndPublish commits the DB transaction and sends the given events with confirmation.
// It also handles and wraps all the errors if there is any.
func CommitAndPublish(tx *gorm.DB, publisher Producer, events ...*events_proto.Event) error {
	if err := utils.HandleDBError(tx.Commit()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	if len(events) > 0 {
		if err := publisher.Send(events...); err != nil {
			// Failed to confirm sent events: rollback the transaction.
			return utils.RollbackTransaction(tx, errors.Wrap(err, "publisher.Send failed"))
		}
	}
	return nil
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
func NewEventFromPayload(
	payload proto.Message,
	eventType events_proto.Event_Type,
	service events_proto.Service,
	fieldMask *field_mask.FieldMask,
) (*events_proto.Event, error) {
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

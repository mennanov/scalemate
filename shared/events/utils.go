package events

import (
	"fmt"
	"regexp"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/utils"
)

const (
	// AccountsSubjectName is a subject name in NATS for the Accounts service events.
	AccountsSubjectName = "accounts"
	// SchedulerSubjectName is a subject name in NATS for the Scheduler service events.
	SchedulerSubjectName = "scheduler"
)

// CommitAndPublish commits the DB transaction and sends the given events with confirmation.
// It also handles and wraps all the errors if there is any.
func CommitAndPublish(tx *gorm.DB, producer Producer, events ...*events_proto.Event) error {
	if len(events) > 0 {
		if err := producer.Send(events...); err != nil {
			// Failed to confirm sent events: rollback the transaction.
			return utils.RollbackTransaction(tx, errors.Wrap(err, "producer.Send failed"))
		}
	}
	// TODO: messages sent to NATS may be processed earlier than they are committed to DB. Need to add a delay in all
	//  the event handlers.
	if err := utils.HandleDBError(tx.Commit()); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

// NewEvent creates a new events_proto.Event instance
func NewEvent(
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

// KeyForEvent returns a key string for the given event.
func KeyForEvent(event *events_proto.Event) string {
	var key string
	switch e := event.Payload.(type) {
	case *events_proto.Event_SchedulerNode:
		key = fmt.Sprintf("scheduler.node.%s", event.Type.String())
		if e.SchedulerNode.Id != 0 {
			key = fmt.Sprintf("%s.%d", key, e.SchedulerNode.Id)
		}

	case *events_proto.Event_SchedulerJob:
		key = fmt.Sprintf("scheduler.job.%s", event.Type.String())
		if e.SchedulerJob.Id != 0 {
			key = fmt.Sprintf("%s.%d", key, e.SchedulerJob.Id)
		}

	case *events_proto.Event_SchedulerTask:
		key = fmt.Sprintf("scheduler.task.%s", event.Type.String())
		if e.SchedulerTask.Id != 0 {
			key = fmt.Sprintf("%s.%d", key, e.SchedulerTask.Id)
		}

	case *events_proto.Event_AccountsUser:
		key = fmt.Sprintf("accounts.user.%s", event.Type.String())
		if e.AccountsUser.Id != 0 {
			key = fmt.Sprintf("%s.%d", key, e.AccountsUser.Id)
		}

	case *events_proto.Event_AccountsNode:
		key = fmt.Sprintf("accounts.node.%s", event.Type.String())
		if e.AccountsNode.Id != 0 {
			key = fmt.Sprintf("%s.%d", key, e.AccountsNode.Id)
		}

	default:
		panic(fmt.Sprintf("unknown event Payload type %T", e))
	}

	return key
}

const msgWaitTimeout = time.Second

// MessagesTestingHandler is used in tests to check if all the expected messages have been received.
type MessagesTestingHandler struct {
	receivedMessageKeys []string
}

// Handle saves all the received message keys.
func (h *MessagesTestingHandler) Handle(event *events_proto.Event) error {
	h.receivedMessageKeys = append(h.receivedMessageKeys, KeyForEvent(event))
	return nil
}

// ExpectMessages waits for the expected messages to be received.
// It returns an error if some messages have not been received within the msgWaitTimeout.
func (h *MessagesTestingHandler) ExpectMessages(keys ...string) error {
	defer func() {
		// Reset the receivedMessageKeys on exit.
		h.receivedMessageKeys = nil
	}()

	var matchedKeys []string
loop:
	for {
		select {
		case <-time.After(msgWaitTimeout):
			break loop

		default:
			if len(matchedKeys) == len(keys) {
				break loop
			}
			for _, keyExpected := range keys {
				re := regexp.MustCompile(keyExpected)
				for _, keyActual := range h.receivedMessageKeys {
					if re.MatchString(keyActual) {
						matchedKeys = append(matchedKeys, keyExpected)
						break
					}
				}
			}
		}
	}
	if len(matchedKeys) != len(keys) {
		return errors.Errorf("expected messages: %s, matched messages: %s", keys, matchedKeys)
	}
	return nil
}

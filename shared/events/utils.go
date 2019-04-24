package events

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"google.golang.org/genproto/protobuf/field_mask"
)

const (
	// AccountsSubjectName is a subject name in NATS for the Accounts service events.
	AccountsSubjectName = "accounts"
	// SchedulerSubjectName is a subject name in NATS for the Scheduler service events.
	SchedulerSubjectName = "scheduler"
)

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

	eventUUID := uuid.New()
	event := &events_proto.Event{
		Uuid:        eventUUID[:],
		Type:        eventType,
		Service:     service,
		PayloadMask: fieldMask,
		CreatedAt:   createdAt,
	}

	switch p := payload.(type) {
	case *scheduler_proto.Container:
		event.Payload = &events_proto.Event_SchedulerContainer{SchedulerContainer: p}

	case *scheduler_proto.Node:
		event.Payload = &events_proto.Event_SchedulerNode{SchedulerNode: p}

	case *accounts_proto.User:
		event.Payload = &events_proto.Event_AccountsUser{AccountsUser: p}

	default:
		return nil, errors.Errorf("unexpected payload type %T", payload)
	}

	return event, nil
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

	case *events_proto.Event_SchedulerContainer:
		key = fmt.Sprintf("scheduler.container.%s", event.Type.String())
		if e.SchedulerContainer.Id != 0 {
			key = fmt.Sprintf("%s.%d", key, e.SchedulerContainer.Id)
		}

	case *events_proto.Event_AccountsUser:
		key = fmt.Sprintf("accounts.user.%s", event.Type.String())
		if e.AccountsUser.Id != 0 {
			key = fmt.Sprintf("%s.%d", key, e.AccountsUser.Id)
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
	mu                  *sync.RWMutex
}

// NewMessagesTestingHandler creates a new instance of MessagesTestingHandler.
func NewMessagesTestingHandler() *MessagesTestingHandler {
	return &MessagesTestingHandler{mu: &sync.RWMutex{}}
}

// Handle saves all the received message keys.
func (h *MessagesTestingHandler) Handle(event *events_proto.Event) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.receivedMessageKeys = append(h.receivedMessageKeys, KeyForEvent(event))
	return nil
}

// ExpectMessages waits for the expected messages to be received.
// It returns an error if some messages have not been received within the msgWaitTimeout.
func (h *MessagesTestingHandler) ExpectMessages(keys ...string) error {
	defer func() {
		// Reset the receivedMessageKeys on exit.
		h.mu.Lock()
		defer h.mu.Unlock()
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
				func() {
					h.mu.RLock()
					defer h.mu.RUnlock()
					for _, keyActual := range h.receivedMessageKeys {
						if re.MatchString(keyActual) {
							matchedKeys = append(matchedKeys, keyExpected)
							break
						}
					}
				}()
			}
		}
	}
	if len(matchedKeys) != len(keys) {
		return errors.Errorf("expected messages: %s, matched messages: %s", keys, matchedKeys)
	}
	return nil
}

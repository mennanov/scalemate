package events_test

import (
	"testing"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/events"
)

func TestKeyForEvent(t *testing.T) {
	testCases := []struct {
		event       *events_proto.Event
		expectedKey string
	}{
		{
			event: &events_proto.Event{
				Type:    events_proto.Event_CREATED,
				Service: events_proto.Service_SCHEDULER,
				Payload: &events_proto.Event_SchedulerContainer{SchedulerContainer: &scheduler_proto.Container{Id: 1}},
			},
			expectedKey: "scheduler.container.CREATED.1",
		},
		{
			event: &events_proto.Event{
				Type:        events_proto.Event_UPDATED,
				Service:     events_proto.Service_SCHEDULER,
				Payload:     &events_proto.Event_SchedulerNode{SchedulerNode: &scheduler_proto.Node{Id: 2}},
				// PayloadMask does not affect the event key.
				PayloadMask: &field_mask.FieldMask{Paths: []string{"status", "connected_at"}},
			},
			expectedKey: "scheduler.node.UPDATED.2",
		},
		{
			event: &events_proto.Event{
				Type:        events_proto.Event_DELETED,
				Service:     events_proto.Service_ACCOUNTS,
				Payload:     &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{Id: 2}},
			},
			expectedKey: "accounts.user.DELETED.2",
		},
		{
			event: &events_proto.Event{
				Type:    events_proto.Event_CREATED,
				Service: events_proto.Service_ACCOUNTS,
				Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{}},
			},
			expectedKey: "accounts.user.CREATED",
		},
	}
	for _, testCase := range testCases {
		key := events.KeyForEvent(testCase.event)
		assert.Equal(t, testCase.expectedKey, key)
	}
}

func TestNewEvent(t *testing.T) {
	userProto := &accounts_proto.User{Id: 42}
	event, err := events.NewEvent(userProto, events_proto.Event_CREATED, events_proto.Service_ACCOUNTS, nil)
	require.NoError(t, err)
	payload, ok := event.Payload.(*events_proto.Event_AccountsUser)
	require.True(t, ok)
	assert.Equal(t, userProto.Id, payload.AccountsUser.Id)
}

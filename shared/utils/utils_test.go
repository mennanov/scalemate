package utils_test

import (
	"testing"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/shared/utils"
)

func TestRoutingKeyFromEvent(t *testing.T) {
	testCases := []struct {
		event       *events_proto.Event
		expectedKey string
	}{
		{
			event: &events_proto.Event{
				Type:    events_proto.Event_CREATED,
				Service: events_proto.Service_SCHEDULER,
				Payload: &events_proto.Event_SchedulerJob{SchedulerJob: &scheduler_proto.Job{}},
			},
			expectedKey: "scheduler.job.created",
		},
		{
			event: &events_proto.Event{
				Type:    events_proto.Event_UPDATED,
				Service: events_proto.Service_SCHEDULER,
				// The actual payload does not affect the routing key, only the mask.
				Payload:     &events_proto.Event_SchedulerJob{SchedulerJob: &scheduler_proto.Job{}},
				PayloadMask: &field_mask.FieldMask{Paths: []string{"status", "connected_at"}},
			},
			expectedKey: "scheduler.job.updated.status.connected_at",
		},
		{
			event: &events_proto.Event{
				Type:    events_proto.Event_CREATED,
				Service: events_proto.Service_ACCOUNTS,
				Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{}},
			},
			expectedKey: "accounts.user.created",
		},
	}
	for _, testCase := range testCases {
		key, err := utils.RoutingKeyFromEvent(testCase.event)
		require.NoError(t, err)
		assert.Equal(t, testCase.expectedKey, key)
	}
}

func TestNewEvent(t *testing.T) {
	userProto := &accounts_proto.User{Id: 42}
	event, err := utils.NewEventFromPayload(userProto, events_proto.Event_CREATED, events_proto.Service_ACCOUNTS, nil)
	require.NoError(t, err)
	payload, ok := event.Payload.(*events_proto.Event_AccountsUser)
	require.True(t, ok)
	assert.Equal(t, userProto.Id, payload.AccountsUser.Id)
}

func TestNewModelProtoFromEvent(t *testing.T) {
	event := &events_proto.Event{
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{
				Id: 1,
			},
		},
	}
	m, err := utils.NewModelProtoFromEvent(event)
	require.NoError(t, err)
	userMsg, ok := m.(*accounts_proto.User)
	require.True(t, ok)
	assert.Equal(t, uint64(1), userMsg.Id)
}

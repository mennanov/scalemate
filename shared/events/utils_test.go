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

func TestNatsSubjectFromEvent(t *testing.T) {
	testCases := []struct {
		event       *events_proto.Event
		expectedKey string
	}{
		{
			event: &events_proto.Event{
				Type:    events_proto.Event_CREATED,
				Service: events_proto.Service_SCHEDULER,
				Payload: &events_proto.Event_SchedulerJob{SchedulerJob: &scheduler_proto.Job{Id: 1}},
			},
			expectedKey: "scheduler.job.CREATED.1",
		},
		{
			event: &events_proto.Event{
				Type:    events_proto.Event_UPDATED,
				Service: events_proto.Service_SCHEDULER,
				// The actual payload does not affect the routing subject, only the mask.
				Payload:     &events_proto.Event_SchedulerJob{SchedulerJob: &scheduler_proto.Job{Id: 2}},
				PayloadMask: &field_mask.FieldMask{Paths: []string{"status", "connected_at"}},
			},
			expectedKey: "scheduler.job.UPDATED.2",
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

func TestNewModelProtoFromEvent(t *testing.T) {
	event := &events_proto.Event{
		Payload: &events_proto.Event_AccountsUser{
			AccountsUser: &accounts_proto.User{
				Id: 1,
			},
		},
	}
	m, err := events.NewModelProtoFromEvent(event)
	require.NoError(t, err)
	userMsg, ok := m.(*accounts_proto.User)
	require.True(t, ok)
	assert.Equal(t, uint64(1), userMsg.Id)
}

package events_test

import (
	"math/rand"
	"os"
	"testing"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func TestNatsProducer_Send(t *testing.T) {
	logger := logrus.New()
	utils.SetLogrusLevelFromEnv(logger)

	sc, err := stan.Connect(os.Getenv("NATS_CLUSTER"), "TestNatsProducer_Send", stan.NatsURL(os.Getenv("NATS_ADDR")))
	require.NoError(t, err)
	defer utils.Close(sc, logger)

	subject := "test_subject"
	producer := events.NewNatsProducer(sc, subject)
	consumer := events.NewNatsConsumer(sc, subject, logrus.New(), stan.DurableName("durable-name"))

	eventsToSend := []*events_proto.Event{
		{
			Type:    events_proto.Event_CREATED,
			Service: events_proto.Service_ACCOUNTS,
			Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{
				Id:       rand.Uint64(),
				Username: "username",
			}},
		},
		{
			Type:    events_proto.Event_UPDATED,
			Service: events_proto.Service_ACCOUNTS,
			Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{
				Id:     rand.Uint64(),
				Banned: true,
			}},
			PayloadMask: &field_mask.FieldMask{Paths: []string{"Banned"}},
		},
	}

	expectedMessageKeys := make([]string, len(eventsToSend))
	for i, event := range eventsToSend {
		key := events.KeyForEvent(event)
		assert.NoError(t, err)
		expectedMessageKeys[i] = key
	}
	// Wait for the expected messages to be received.
	received, err := events.WaitForMessages(consumer, logger, expectedMessageKeys...)
	require.NoError(t, err)

	err = producer.Send(eventsToSend...)
	require.NoError(t, err)
	assert.NoError(t, <-received)
}

package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/events/events_proto"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/protobuf/field_mask"
)

// AMQPEnvConf maps to the name of the env variable with the AMQP address to connect to.
var AMQPEnvConf = utils.AMQPEnvConf{
	Addr: "SHARED_AMQP_ADDR",
}

func TestAMQPPublisher_Send(t *testing.T) {
	exchangeName := "publisher_send_test"
	conn, err := utils.ConnectAMQPFromEnv(AMQPEnvConf)
	require.NoError(t, err)
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()

	err = utils.AMQPExchangeDeclare(ch, exchangeName)
	require.NoError(t, err)

	messages, err := utils.SetUpAMQPTestConsumer(conn, exchangeName)
	require.NoError(t, err)

	receivedMessages := make([]amqp.Delivery, 0)
	go func() {
		for msg := range messages {
			receivedMessages = append(receivedMessages, msg)
		}
	}()

	publisher, err := events.NewAMQPPublisher(conn, exchangeName)
	require.NoError(t, err)

	event := &events_proto.Event{
		Type:    events_proto.Event_CREATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{
			Id:       1,
			Username: "username",
		}},
	}

	err = publisher.Send(event)
	require.NoError(t, err)
	// Wait till all the messages are received.
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, 1, len(receivedMessages))
	routingKey, err := events.RoutingKeyFromEvent(event)
	require.NoError(t, err)
	assert.Equal(t, routingKey, receivedMessages[0].RoutingKey)
}

func TestAMQPPublisher_SendWithConfirmation(t *testing.T) {
	exchangeName := "publisher_send_test_with_confirmation"
	conn, err := utils.ConnectAMQPFromEnv(AMQPEnvConf)
	require.NoError(t, err)
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()

	err = utils.AMQPExchangeDeclare(ch, exchangeName)
	require.NoError(t, err)

	messages, err := utils.SetUpAMQPTestConsumer(conn, exchangeName)
	require.NoError(t, err)

	receivedMessages := make([]amqp.Delivery, 0)
	go func() {
		for msg := range messages {
			receivedMessages = append(receivedMessages, msg)
		}
	}()

	publisher, err := events.NewAMQPPublisher(conn, exchangeName)
	require.NoError(t, err)

	eventsToSend := []*events_proto.Event{
		{
			Type:    events_proto.Event_CREATED,
			Service: events_proto.Service_ACCOUNTS,
			Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{
				Id:       1,
				Username: "username",
			}},
		},
		{
			Type:    events_proto.Event_UPDATED,
			Service: events_proto.Service_ACCOUNTS,
			Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{
				Id:       1,
				Username: "username",
			}},
			PayloadMask: &field_mask.FieldMask{Paths: []string{"banned"}},
		},
	}

	ctx := context.Background()

	err = publisher.SendWithConfirmation(ctx, eventsToSend...)
	require.NoError(t, err)
	// Wait till all the messages are received.
	time.Sleep(time.Millisecond * 200)

	assert.Equal(t, 2, len(receivedMessages))
	expectedRoutingKeys := make([]string, 2)
	for i, event := range eventsToSend {
		rk, err := events.RoutingKeyFromEvent(event)
		assert.NoError(t, err)
		expectedRoutingKeys[i] = rk
	}
	actualRoutingKeys := make([]string, 2)
	for i, msg := range receivedMessages {
		actualRoutingKeys[i] = msg.RoutingKey
	}
	assert.Equal(t, expectedRoutingKeys, actualRoutingKeys)
}

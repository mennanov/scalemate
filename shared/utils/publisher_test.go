package utils_test

import (
	"testing"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/shared/utils"
)

// AMQPEnvConf maps to the name of the env variable with the AMQP address to connect to.
var AMQPEnvConf = utils.AMQPEnvConf{
	Addr: "SHARED_AMQP_ADDR",
}

func TestAMQPPublisher_SendAsync(t *testing.T) {
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

	publisher, err := utils.NewAMQPPublisher(conn, exchangeName)
	require.NoError(t, err)

	event := &events_proto.Event{
		Type:    events_proto.Event_CREATED,
		Service: events_proto.Service_ACCOUNTS,
		Payload: &events_proto.Event_AccountsUser{AccountsUser: &accounts_proto.User{
			Id:       1,
			Username: "username",
		}},
	}

	err = publisher.SendAsync(event)
	require.NoError(t, err)
	routingKey, err := utils.RoutingKeyFromEvent(event)
	require.NoError(t, err)
	utils.WaitForMessages(messages, routingKey)
}

func TestAMQPPublisher_Send(t *testing.T) {
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

	publisher, err := utils.NewAMQPPublisher(conn, exchangeName)
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

	err = publisher.Send(eventsToSend...)
	require.NoError(t, err)
	expectedRoutingKeys := make([]string, 2)
	for i, event := range eventsToSend {
		rk, err := utils.RoutingKeyFromEvent(event)
		assert.NoError(t, err)
		expectedRoutingKeys[i] = rk
	}
	// Wait till all the messages are received.
	utils.WaitForMessages(messages, expectedRoutingKeys...)
}

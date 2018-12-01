package utils

import (
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// SetUpAMQPTestConsumer declares an AMQP exchange, declares a queue (with binding) to consume all messages from this
// exchange.
func SetUpAMQPTestConsumer(conn *amqp.Connection, exchangeName string) (<-chan amqp.Delivery, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	queue, err := channel.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, err
	}
	if err := channel.QueueBind(queue.Name, "#", exchangeName, false, nil); err != nil {
		return nil, err
	}

	messages, err := channel.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// WaitForMessages waits for the messages and matches their routing keys with the given keys.
// The function returns once all the provided keys have matched.
func WaitForMessages(messages <-chan amqp.Delivery, keys ...string) {
	go func() {
		for msg := range messages {
			logrus.Debugf("received AMQP message %s", msg.RoutingKey)
			for i := range keys {
				re := regexp.MustCompile(keys[i])
				if re.MatchString(msg.RoutingKey) {
					// Remove the key that matched from the keys.
					keys = append(keys[:i], keys[i+1:]...)
					if len(keys) == 0 {
						return
					}
					break
				}
			}
		}
	}()
}

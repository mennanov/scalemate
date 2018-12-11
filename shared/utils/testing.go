package utils

import (
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// WaitForMessages waits for the messages and matches their routing keys with the given keys.
// The function returns once all the provided keys have matched.
func WaitForMessages(messages <-chan amqp.Delivery, keys ...string) {
	allMessagesReceived := make(chan struct{})
	go func(c chan struct{}) {
		for msg := range messages {
			logrus.Debugf("Received AMQP message %s", msg.RoutingKey)
			for i := range keys {
				re := regexp.MustCompile(keys[i])
				if re.MatchString(msg.RoutingKey) {
					// Remove the key that matched from the keys.
					keys = append(keys[:i], keys[i+1:]...)
					if len(keys) == 0 {
						c <- struct{}{}
						return
					}
					break
				}
			}
		}
	}(allMessagesReceived)

	// Wait until all messages are received.
	<-allMessagesReceived
}

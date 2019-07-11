package events

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventHandler defines an interface for processing events.
type EventHandler interface {
	// Handle processes the given event.
	Handle(context.Context, *events_proto.Event) error
}

// StanMsgHandler returns a stan.MsgHandler for the given event handlers.
func StanMsgHandler(ctx context.Context, logger *logrus.Logger, ackRetryLimit int, handlers ...EventHandler) stan.MsgHandler {
	return func(msg *stan.Msg) {
		logger.Debug("received message: %s", msg.String())

		eventsMessage := &events_proto.Message{}
		if err := proto.Unmarshal(msg.Data, eventsMessage); err != nil {
			// This is a non-retryable error. Acknowledge the message since it will never succeed anyway.
			logger.Error(errors.Wrap(err, "proto.Unmarshal failed"))
			if err := ackMsg(msg, ackRetryLimit, logger); err != nil {
				logger.Error(err)
			}
			return
		}

		ack := true
		wg := new(sync.WaitGroup)
		for _, event := range eventsMessage.Events {
			for _, handler := range handlers {
				go func(handler EventHandler) {
					wg.Add(1)
					defer wg.Done()
					if err := handler.Handle(ctx, event); err != nil {
						ack = false
						logger.WithError(err).Error("event handler failed")
					}
				}(handler)
			}
		}

		wg.Wait()

		if ack {
			// Manually acknowledge the message if there were no errors.
			// Unacknowledged messages will be redelivered after the AckTimeout.
			if err := ackMsg(msg, ackRetryLimit, logger); err != nil {
				logger.Error(err)
			}
		}
	}
}

func ackMsg(msg *stan.Msg, retryLimit int, logger *logrus.Logger) error {
	var err error
	for i := 0; i <= retryLimit; i++ {
		err = msg.Ack()
		if err == nil {
			break
		}

		if i < retryLimit {
			logger.WithError(err).WithField("msg", msg.String()).Info("retrying ack in 1 second")
			time.Sleep(time.Second)
		}
	}
	return errors.Wrap(err, "failed to acknowledge a message")
}

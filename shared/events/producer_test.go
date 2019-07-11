package events_test

import (
	"os"
	"testing"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/testutils"
	"github.com/mennanov/scalemate/shared/utils"
)

func TestNatsProducer_Send(t *testing.T) {
	logger := logrus.New()
	utils.SetLogrusLevelFromEnv(logger)

	sc, err := stan.Connect(os.Getenv("NATS_CLUSTER"), "TestNatsProducer_Send", stan.NatsURL(os.Getenv("NATS_ADDR")))
	require.NoError(t, err)
	defer utils.Close(sc, logger)

	subject := "test_subject"
	producer := events.NewNatsProducer(sc, subject, 5)
	wait := testutils.ExpectMessages(sc, subject, logger, "Event_SchedulerNodeConnected")

	err = producer.Send(&events_proto.Event{
		Payload: &events_proto.Event_SchedulerNodeConnected{
			SchedulerNodeConnected: &scheduler_proto.NodeConnectedEvent{
				Node: new(scheduler_proto.Node),
			},
		},
	})
	require.NoError(t, err)
	require.NoError(t, wait(time.Second))
}

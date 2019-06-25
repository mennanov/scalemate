package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/accounts/conf"
	"github.com/mennanov/scalemate/accounts/handlers"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// handleEventsCmd represents the handle_events command.
var handleEventsCmd = &cobra.Command{
	Use:   "handle_events",
	Short: "Handle events consumed from a message broker. Client ID must be provided.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.StandardLogger()
		logger.SetLevel(logrus.Level(verbosity))

		db, err := utils.ConnectDBFromEnv(conf.AccountsConf.DBUrl)
		if err != nil {
			logger.Fatalf("Failed to connect to database: %s", err.Error())
		}
		defer utils.Close(db, logger)

		clientID := args[0]
		sc, err := stan.Connect(conf.AccountsConf.NatsClusterName, clientID, stan.NatsURL(conf.AccountsConf.NatsAddr))
		if err != nil {
			logger.Fatalf("stan.Connect failed: %s", err.Error())
		}
		defer utils.Close(sc, logger)

		producer := events.NewNatsProducer(sc, events.AccountsSubjectName, 5)
		// NatsQueueConsumer is used to make the events processing scalable (round-robin delivery).
		consumer := events.NewNatsQueueConsumer(sc, events.SchedulerSubjectName,
			"accounts-events-handlers-queue", producer, db, logger, 3,
			stan.DurableName("accounts-events-handlers"),
			stan.AckWait(time.Second*5))

		subscription, err := consumer.Consume(handlers.NewSchedulerNodeCreatedHandler(logger))
		if err != nil {
			logger.Fatalf("consumer.Consume failed: %s", err.Error())
		}
		defer utils.Close(subscription, logger)

		terminate := make(chan os.Signal)
		signal.Notify(terminate, os.Interrupt, syscall.SIGTERM)

		logger.Info("Started handling events...")
		<-terminate
		logger.Info("Stopped handling events.")
	},
}

func init() {
	rootCmd.AddCommand(handleEventsCmd)
}

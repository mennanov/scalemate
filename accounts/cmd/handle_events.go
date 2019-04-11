package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
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
	Short: "Handle events consumed from a message broker.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.StandardLogger()
		logger.SetLevel(logrus.Level(verbosity))

		db, err := utils.ConnectDBFromEnv(conf.AccountsConf.DBUrl)
		if err != nil {
			logger.Fatalf("Failed to connect to database: %s", err.Error())
			os.Exit(1) //revive:disable-line:deep-exit
		}
		defer utils.Close(db, logger)

		sc, err := stan.Connect(
			conf.AccountsConf.NatsClusterName,
			fmt.Sprintf("accounts-events-handler-%s", uuid.New().String()),
			stan.NatsURL(conf.AccountsConf.NatsAddr))
		if err != nil {
			logger.Fatalf("stan.Connect failed: %s", err.Error())
			os.Exit(1) //revive:disable-line:deep-exit
		}
		defer utils.Close(sc, logger)

		producer := events.NewNatsProducer(sc, events.AccountsSubjectName)
		// NatsQueueConsumer is used to make the events processing scalable (round-robin delivery).
		consumer := events.NewNatsQueueConsumer(sc, events.SchedulerSubjectName,
			"accounts-events-handlers-queue", logger,
			stan.DurableName("accounts-events-handlers"),
			stan.AckWait(time.Second*5))

		subscription, err := consumer.Consume(handlers.NewSchedulerNodeCreatedHandler(db, producer, logger))
		if err != nil {
			logger.Fatalf("consumer.Consume failed: %s", err.Error())
			os.Exit(1) //revive:disable-line:deep-exit
		}
		defer utils.Close(subscription, logger)

		go func() {
			for err := range subscription.Errors() {
				logger.WithError(err).Error("error occurred while processing an event message")
			}
		}()

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

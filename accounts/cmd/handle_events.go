package cmd

import (
	"context"
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

		ctx, cancel := context.WithCancel(context.Background())
		// QueueSubscribe is used to make the events processing scalable (round-robin delivery).
		subscription, err := sc.QueueSubscribe(events.SchedulerSubjectName, "accounts-events-handlers-queue",
			events.StanMsgHandler(ctx, logger, 5, handlers.NewSchedulerNodeCreatedHandler(db)),
			stan.DurableName("accounts-events-handlers"),
			stan.AckWait(time.Second*5))
		if err != nil {
			logger.WithError(err).Fatal("failed to subscribe to NATS")
		}
		defer utils.Close(subscription, logger)

		terminate := make(chan os.Signal)
		signal.Notify(terminate, os.Interrupt, syscall.SIGTERM)

		logger.Info("Started handling events...")
		<-terminate
		cancel()
		logger.Info("Stopped handling events.")
	},
}

func init() {
	rootCmd.AddCommand(handleEventsCmd)
}

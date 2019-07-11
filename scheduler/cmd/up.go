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
	"google.golang.org/grpc/credentials"

	"github.com/mennanov/scalemate/scheduler/conf"
	"github.com/mennanov/scalemate/scheduler/frontend"
	"github.com/mennanov/scalemate/scheduler/handlers"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.StandardLogger()
		utils.SetLogrusLevelFromEnv(logger)

		creds, err := credentials.NewServerTLSFromFile(conf.SchdulerConf.TLSCertFile, conf.SchdulerConf.TLSKeyFile)
		if err != nil {
			logger.WithError(err).Fatal("NewServerTLSFromFile failed")
		}

		// Connect to DB.
		db, err := utils.ConnectDBFromEnv(conf.SchdulerConf.DBUrl)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to DB")
		}
		defer utils.Close(db, nil)

		clientID := args[0]
		sc, err := stan.Connect(conf.SchdulerConf.NatsClusterName, clientID, stan.NatsURL(conf.SchdulerConf.NatsAddr))
		if err != nil {
			logger.WithError(err).Fatal("stan.Connect failed")
		}
		defer utils.Close(sc, logger)

		producer := events.NewNatsProducer(sc, events.SchedulerSubjectName, 5)

		ctx, ctxCancel := context.WithCancel(context.Background())

		subscription, err := sc.Subscribe(events.SchedulerSubjectName,
			events.StanMsgHandler(ctx, logger, 5, handlers.NewIndependentHandlers(db, producer, logger, 5)...),
			stan.AckWait(time.Second*15))
		if err != nil {
			logger.WithError(err).Fatal("failed to subscribe to NATS")
		}
		defer utils.Close(subscription, logger)

		shutdown := make(chan os.Signal)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-shutdown
			ctxCancel()
		}()

		frontend.NewSchedulerServer(
			frontend.WithLogger(logger),
			frontend.WithDBConnection(db),
			frontend.WithProducer(producer),
			frontend.WithClaimsInjector(auth.NewJWTClaimsInjector(conf.SchdulerConf.JWTSecretKey)),
		).Serve(ctx, grpcAddr, creds)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}

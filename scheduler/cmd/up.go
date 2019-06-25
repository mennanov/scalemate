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
	"github.com/mennanov/scalemate/scheduler/server"
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
		consumer := events.NewNatsConsumer(sc, events.SchedulerSubjectName, producer, db, logger, 5,
			3, stan.DurableName("scheduler-in-service-event-handlers"), stan.AckWait(time.Second*15))

		subscription, err := consumer.Consume()
		if err != nil {
			logger.WithError(err).Fatal("failed to start NATS consumer")
		}
		defer utils.Close(subscription, logger)

		shutdown := make(chan os.Signal)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		ctx, ctxCancel := context.WithCancel(context.Background())

		go func() {
			<-shutdown
			ctxCancel()
		}()

		server.NewSchedulerServer(
			server.WithLogger(logger),
			server.WithDBConnection(db),
			server.WithProducer(producer),
			server.WithClaimsInjector(auth.NewJWTClaimsInjector(conf.SchdulerConf.JWTSecretKey)),
		).Serve(ctx, grpcAddr, creds)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}

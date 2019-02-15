package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to DB.
		db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
		if err != nil {
			logrus.WithError(err).Error("utils.ConnectDBFromEnv failed")
			return
		}
		defer utils.Close(db)

		// Connect to AMQP.
		amqpConnection, err := utils.ConnectAMQPFromEnv(server.AMQPEnvConf)
		if err != nil {
			logrus.WithError(err).Error("utils.ConnectAMQPFromEnv failed")
			return
		}
		defer utils.Close(amqpConnection)

		logger := logrus.StandardLogger()
		utils.SetLogrusLevelFromEnv(logger)

		producer, err := events.NewAMQPProducer(amqpConnection, events.SchedulerAMQPExchangeName)
		if err != nil {
			logger.WithError(err).Error("events.NewAMQPProducer failed")
			return
		}

		claimsInjector, err := server.JWTClaimsInjectorFromEnv(server.SchedulerEnvConf)
		if err != nil {
			logger.WithError(err).Error("server.JWTClaimsInjectorFromEnv failed")
			return
		}

		schedulerServer, err := server.NewSchedulerServer(
			server.WithLogger(logger),
			server.WithDBConnection(db),
			server.WithProducer(producer),
			server.WithAMQPConsumers(amqpConnection),
			server.WithClaimsInjector(claimsInjector),
		)
		if err != nil {
			logger.WithError(err).Error("server.NewSchedulerServer failed")
			return
		}
		defer utils.Close(schedulerServer)

		shutdown := make(chan os.Signal)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

		schedulerServer.Serve(grpcAddr, shutdown)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}

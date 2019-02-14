package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/scheduler/server"
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

		schedulerServer, err := server.NewSchedulerServer(
			server.WithLogger(logger),
			server.WithDBConnection(db),
			server.WithAMQPProducer(amqpConnection),
			server.WithAMQPConsumers(amqpConnection),
			server.WithClaimsInjector(server.SchedulerEnvConf),
		)
		if err != nil {
			logrus.WithError(err).Error("server.NewSchedulerServer failed")
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

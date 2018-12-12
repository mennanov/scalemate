package cmd

import (
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

		schedulerServer, err := server.NewSchedulerServerFromEnv(server.SchedulerEnvConf, db, amqpConnection)
		if err != nil {
			logrus.WithError(err).Error("server.NewSchedulerServerFromEnv failed")
			return
		}
		defer utils.Close(schedulerServer)

		schedulerServer.Serve(grpcAddr)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}

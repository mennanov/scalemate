package cmd

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/utils"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
		if err != nil {
			logrus.WithError(err).Error("utils.ConnectDBFromEnv failed")
			return
		}
		defer utils.Close(db)

		amqpConnection, err := utils.ConnectAMQPFromEnv(server.AMQPEnvConf)
		if err != nil {
			logrus.WithError(err).Error("utils.ConnectAMQPFromEnv failed")
			return
		}
		defer utils.Close(amqpConnection)

		accountsServer, err := server.NewAccountServerFromEnv(server.AccountsEnvConf, db, amqpConnection)
		if err != nil {
			fmt.Printf("Failed to start gRPC server: %+v\n", err)
			return
		}
		defer utils.Close(accountsServer)

		utils.SetLogrusLevelFromEnv()

		accountsServer.Serve(grpcAddr)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}

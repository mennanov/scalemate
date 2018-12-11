package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/accounts/server"
	"github.com/mennanov/scalemate/shared/utils"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Run: func(cmd *cobra.Command, args []string) {
		accountsServer, err := server.NewAccountServerFromEnv(server.AccountsEnvConf, server.DBEnvConf, server.AMQPEnvConf)
		if err != nil {
			fmt.Printf("Failed to start gRPC server: %+v\n", err)
			return
		}
		defer utils.Close(accountsServer)

		server.Serve(grpcAddr, accountsServer)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}

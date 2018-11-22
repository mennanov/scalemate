package cmd

import (
	"fmt"

	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/spf13/cobra"
)

var grpcAddr string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "up runs a gRPC service",
	Run: func(cmd *cobra.Command, args []string) {
		schedulerServer, err := server.NewSchedulerServerFromEnv(server.SchedulerEnvConf)
		defer utils.Close(schedulerServer)

		if err != nil {
			panic(fmt.Sprintf("Failed to create gRPC server from env: %s", err.Error()))
		}

		server.Serve(grpcAddr, schedulerServer)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringVarP(&grpcAddr, "addr", "a", "0.0.0.0:8000", "GRPC server net addr")
}

package cmd

import (
	"fmt"

	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/spf13/cobra"
)

// rollbackCmd represents the rollback command
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback a migration",
	Long:  "Rollback sequentially to the given migration by ID or the last migration if ID is not provided",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
		if err != nil {
			fmt.Printf("Failed to connect to DB: %s", err.Error())
		}

		if err := migrations.RollbackMigrations(db, args[0]); err != nil {
			fmt.Printf("Failed to rollback migrations: %v\n", err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

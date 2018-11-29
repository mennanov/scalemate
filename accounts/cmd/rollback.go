package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/accounts/migrations"
	"github.com/mennanov/scalemate/shared/utils"
)

// rollbackCmd represents the rollback command
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback a migration",
	Long:  "Rollback sequentially to the given migration by ID or the last migration if ID is not provided",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := utils.ConnectDBFromEnv(utils.DBEnvConf{})
		if err != nil {
			fmt.Printf("Failed to connect to database: %+v\n", err)
			return
		}
		if err := migrations.RollbackMigrations(db, args[0]); err != nil {
			fmt.Printf("Failed to rollback migrations: %v\n", err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

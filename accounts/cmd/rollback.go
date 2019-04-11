package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/accounts/conf"
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
		logger := logrus.StandardLogger()
		logger.SetLevel(logrus.Level(verbosity))
		db, err := utils.ConnectDBFromEnv(conf.AccountsConf.DBUrl)
		if err != nil {
			logger.WithError(err).Fatalf("Failed to connect to database: %+v\n", err)
			os.Exit(1) //revive:disable-line:deep-exit
		}
		if err := migrations.RollbackMigrations(db, args[0]); err != nil {
			logger.WithError(err).Fatalf("Failed to rollback migrations: %v\n", err)
			os.Exit(1) //revive:disable-line:deep-exit
		}
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/accounts/conf"
	"github.com/mennanov/scalemate/accounts/migrations"
	"github.com/mennanov/scalemate/shared/utils"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply all unapplied migrations",
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.StandardLogger()
		logger.SetLevel(logrus.Level(verbosity))

		logger.Info("Running migrations...")
		db, err := utils.ConnectDBFromEnv(conf.AccountsConf.DBUrl)
		if err != nil {
			logger.WithError(err).Fatalf("Failed to connect to database: %+v\n", err)
			os.Exit(1) //revive:disable-line:deep-exit
		}
		if err := migrations.RunMigrations(db); err != nil {
			logger.WithError(err).Fatalf("Failed to apply migrations: %v\n", err.Error())
			os.Exit(1) //revive:disable-line:deep-exit
		}
		logger.Info("Migrations applied.")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

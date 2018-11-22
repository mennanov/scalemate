package cmd

import (
	"fmt"

	"github.com/mennanov/scalemate/scheduler/migrations"
	"github.com/mennanov/scalemate/scheduler/server"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply all unapplied migrations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running migrations...")
		db, err := utils.ConnectDBFromEnv(server.DBEnvConf)
		if err != nil {
			fmt.Printf("Failed to connect to DB: %s", err.Error())
		}

		db.LogMode(true)

		if err := migrations.RunMigrations(db); err != nil {
			fmt.Printf("Failed to apply migrations: %v\n", err.Error())
		}
		fmt.Println("migrations applied.")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

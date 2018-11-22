package cmd

import (
	"fmt"

	"github.com/mennanov/scalemate/accounts/migrations"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply all unapplied migrations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running migrations...")
		db, err := utils.ConnectDBFromEnv(utils.DBEnvConf{})
		if err != nil {
			fmt.Printf("Failed to connect to database: %+v\n", err)
			return
		}
		if err := migrations.RunMigrations(db); err != nil {
			fmt.Printf("Failed to apply migrations: %v\n", err.Error())
			return
		}
		fmt.Println("migrations applied.")
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

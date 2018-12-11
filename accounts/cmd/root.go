package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Scalemate.io accounts service",
}

// Execute runs the Cobra root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1) //revive:disable-line:deep-exit
	}
}

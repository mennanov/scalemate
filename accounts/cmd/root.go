package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var verbosity uint32

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

func init() {
	rootCmd.Flags().Uint32VarP(&verbosity, "verbosity", "v", uint32(logrus.WarnLevel), "Verbosity level [0-6]")
}

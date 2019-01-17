// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/mennanov/scalemate/client/accounts"
	"github.com/mennanov/scalemate/client/scheduler"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "scalemate",
	Short: "Run Docker containers on the premises provided by Scalemate.io",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogger(verbose)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1) //revive:disable-line:deep-exit
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.scalemate.yaml)")
	rootCmd.PersistentFlags().
		BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.PersistentFlags().
		StringVar(&accounts.ServiceAddr, "accounts_addr", "localhost:8000",
			"Scalemate.io Accounts service tcp address")
	viper.BindPFlag("accounts_addr", rootCmd.PersistentFlags().Lookup("accounts_addr"))
	viper.SetDefault("accounts_addr", "localhost:8000")

	rootCmd.PersistentFlags().
		StringVar(&scheduler.ServiceAddr, "scheduler_addr", "localhost:8001",
			"Scalemate.io Scheduler service tcp address")
	viper.BindPFlag("scheduler_addr", rootCmd.PersistentFlags().Lookup("scheduler_addr"))
	viper.SetDefault("scheduler_addr", "localhost:8001")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1) //revive:disable-line:deep-exit
		}

		// Search config in home directory with name ".scalemate" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".scalemate")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func setupLogger(verbose bool) {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}
}

func userInput(msg string, to *string) error {
	fmt.Printf("%s: ", msg)
	_, err := fmt.Scanln(to)
	return err
}

func userPasswordInput(msg string, to *string) error {
	fmt.Printf("%s: ", msg)
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}
	*to = string(bytePassword)
	fmt.Print("\n")
	return nil
}

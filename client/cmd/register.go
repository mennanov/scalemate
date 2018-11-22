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
	"github.com/mennanov/scalemate/client/accounts"
	"github.com/spf13/cobra"
	"os"
)

var (
	registerUsername string
	registerEmail    string
	registerPassword string
)

// registerCmd represents the register command
var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "RegisterController a new account at Scalemate.io",
	Run: func(cmd *cobra.Command, args []string) {
		if registerUsername == "" {
			if err := userInput("Username", &registerUsername); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return
			}
		}
		if registerEmail == "" {
			if err := userInput("Email", &registerEmail); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return
			}
		}
		if registerPassword == "" {
			if err := userPasswordInput("Password", &registerPassword); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return
			}
		}

		accounts.RegisterView(os.Stdout, os.Stderr,
			accounts.RegisterController(
				accounts.NewAccountsClient(), registerUsername, registerEmail, registerPassword))
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// registerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	registerCmd.Flags().StringVarP(&registerUsername, "username", "u", "", "username")
	registerCmd.Flags().StringVarP(&registerEmail, "email", "m", "", "email")
	registerCmd.Flags().StringVarP(&registerPassword, "password", "p", "", "password")
}

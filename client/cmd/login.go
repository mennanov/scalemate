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
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/client/accounts"
	"github.com/mennanov/scalemate/shared/client"
)

var (
	loginUsername string
	loginPassword string
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Scalemate.io",
	Run: func(cmd *cobra.Command, args []string) {
		if loginUsername == "" {
			if err := userInput("Username", &loginUsername); err != nil {
				logger.Error(err.Error())
				return
			}
		}
		if loginPassword == "" {
			if err := userPasswordInput("Password", &loginPassword); err != nil {
				logger.Error(err.Error())
				return
			}
		}
		_, err := accounts.LoginController(
			client.NewAccountsClient(accountsServiceAddr), loginUsername, loginPassword)
		accounts.LoginView(logger, err)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	loginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "username")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "password")
}

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
	changePasswordPassword string
	changePasswordConfirm  string
)

// changePasswordCmd represents the changePassword command
var changePasswordCmd = &cobra.Command{
	Use:   "changePassword",
	Short: "Change a password for the currently logged in user",
	Run: func(cmd *cobra.Command, args []string) {
		if changePasswordPassword == "" {
			if err := userPasswordInput("Password", &changePasswordPassword); err != nil {
				logger.Error(err.Error())
				return
			}
			if err := userPasswordInput("Same password again", &changePasswordConfirm); err != nil {
				logger.Error(err.Error())
				return
			}
			if changePasswordPassword != changePasswordConfirm {
				logger.Error("Passwords do not match.")
				return
			}
		}
		accounts.ChangePasswordView(logger,
			accounts.ChangePasswordController(client.NewAccountsClient(accountsServiceAddr), changePasswordPassword))
	},
}

func init() {
	rootCmd.AddCommand(changePasswordCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// changePasswordCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// changePasswordCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	changePasswordCmd.Flags().StringVarP(&changePasswordPassword, "password", "p", "", "password")
}

// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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

	"github.com/spf13/cobra"
)

// jobsCmd represents the jobs command
var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Create, list and get jobs",
}

// jobsCreateCmd represents the job creation command
var jobsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Job",
	Long: `Create a new Job entity which will be scheduled immediately. 
	Once the Job is scheduled a corresponding Task entity is created.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create called")
		dockerImage := args[0]
		var dockerCommand []string
		if len(args) > 1 {
			dockerCommand = args[1:]
		}
		fmt.Println("image:", dockerImage, "command", dockerCommand)
	},
}

func init() {
	jobsCmd.AddCommand(jobsCreateCmd)

	rootCmd.AddCommand(jobsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// jobsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	jobsCreateCmd.Flags().BoolP("daemon", "d", false,
		"Don't wait for an established p2p connection to run a container")

}

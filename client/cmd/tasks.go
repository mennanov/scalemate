package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/client/scheduler"
	"github.com/mennanov/scalemate/shared/client"
)

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Get, list and iterate tasks",
}

func init() {
	tasksCmd.AddCommand(tasksGetCmd)

	rootCmd.AddCommand(tasksCmd)
}

var tasksGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get an existing Task",
	Long:    `Get an existing Task by its ID.`,
	Args:    cobra.ExactArgs(1),
	Example: `> scalemate tasks get 42`,
	Run: func(cmd *cobra.Command, args []string) {
		taskID, err := strconv.Atoi(args[0])
		if err != nil || taskID <= 0 {
			fmt.Printf("invalid Task ID: %s\n", args[0])
			return
		}
		task, err := scheduler.GetTaskController(
			client.NewAccountsClient(accountsServiceAddr),
			client.NewSchedulerClient(schedulerServiceAddr),
			uint64(taskID))
		scheduler.JSONPbView(logger, os.Stdout, task, err)
	},
}

package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/client/scheduler"
	"github.com/mennanov/scalemate/shared/client"
)

var (
	tasksListCmdFlagValues         = scheduler.ListTasksCmdFlags{}
	tasksIterateCmdIncludeExisting bool
)

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Get, list and iterate tasks",
}

func init() {
	tasksCmd.AddCommand(tasksGetCmd)
	tasksCmd.AddCommand(tasksListCmd)
	tasksCmd.AddCommand(tasksCancelCmd)
	tasksCmd.AddCommand(tasksIterateCmd)

	rootCmd.AddCommand(tasksCmd)

	// tasksListCmd flags.
	tasksListCmd.Flags().IntSliceVarP(&tasksListCmdFlagValues.Status, "status", "s", nil,
		fmt.Sprintf("Task statuses, options: %s", enumOptions(scheduler_proto.Task_Status_name)))
	tasksListCmd.Flags().Int32VarP(&tasksListCmdFlagValues.Ordering, "order", "o", 0,
		fmt.Sprintf("Ordering, options: %s", enumOptions(scheduler_proto.ListTasksRequest_Ordering_name)))
	tasksListCmd.Flags().Uint32VarP(&tasksListCmdFlagValues.Limit, "limit", "l", 10, "Tasks limit")
	tasksListCmd.Flags().Uint32Var(&tasksListCmdFlagValues.Offset, "offset", 0, "Tasks offset")
	tasksListCmd.Flags().BoolVarP(&tasksIterateCmdIncludeExisting, "existing", "e", false,
		"Include already existing Tasks")
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
			client.NewSchedulerFrontEndClient(schedulerServiceAddr),
			uint64(taskID))
		scheduler.JSONPbView(logger, os.Stdout, task, err)
	},
}

var tasksListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List existing Tasks",
	Long:    `List existing Tasks for the currently authenticated user that satisfy the criteria.`,
	Example: `> scalemate tasks list 42 56 -s 1 -o 2 -l 50`,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jobIDs := make([]uint64, len(args))
		for i, a := range args {
			jobID, err := strconv.Atoi(a)
			if err != nil || jobID <= 0 {
				fmt.Printf("invalid Job ID: %s\n", a)
				return
			}
			jobIDs[i] = uint64(jobID)
		}

		response, err := scheduler.ListTasksController(
			client.NewAccountsClient(accountsServiceAddr),
			client.NewSchedulerFrontEndClient(schedulerServiceAddr),
			jobIDs,
			&tasksListCmdFlagValues,
		)
		scheduler.JSONPbView(logger, os.Stdout, response, err)
	},
}

var tasksCancelCmd = &cobra.Command{
	Use:     "cancel",
	Short:   "Cancel an existing Task",
	Long:    `Cancel an existing Task by its ID.`,
	Args:    cobra.ExactArgs(1),
	Example: `> scalemate tasks cancel 42`,
	Run: func(cmd *cobra.Command, args []string) {
		taskID, err := strconv.Atoi(args[0])
		if err != nil || taskID <= 0 {
			fmt.Printf("invalid Task ID: %s\n", args[0])
			return
		}
		task, err := scheduler.CancelTaskController(
			client.NewAccountsClient(accountsServiceAddr),
			client.NewSchedulerFrontEndClient(schedulerServiceAddr),
			uint64(taskID))
		scheduler.JSONPbView(logger, os.Stdout, task, err)
	},
}

var tasksIterateCmd = &cobra.Command{
	Use:     "iter",
	Short:   "Iterate over Tasks",
	Long:    `Iterate over existing and new Tasks by Job ID. The Tasks are received upon creation.`,
	Args:    cobra.ExactArgs(1),
	Example: `> scalemate tasks iter 42`,
	Run: func(cmd *cobra.Command, args []string) {
		jobID, err := strconv.Atoi(args[0])
		if err != nil || jobID <= 0 {
			fmt.Printf("invalid Job ID: %s\n", args[0])
			return
		}
		client, err := scheduler.IterateTasksController(
			client.NewAccountsClient(accountsServiceAddr),
			client.NewSchedulerFrontEndClient(schedulerServiceAddr),
			uint64(jobID),
			tasksIterateCmdIncludeExisting)

		// TODO: figure out how to handle ctrl+c. Do we need to intercept signals?
		for {
			task, err := client.Recv()
			if err == io.EOF {
				// The stream is closed.
				break
			}
			scheduler.JSONPbView(logger, os.Stdout, task, err)
		}
	},
}

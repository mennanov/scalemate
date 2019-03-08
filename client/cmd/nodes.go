package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/client/scheduler"
	"github.com/mennanov/scalemate/shared/client"
)

var (
	listNodesCmdFlagValues = scheduler.ListNodesCmdFlags{}
)

// nodesCmd represents the nodes command
var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "List and get nodes",
}

func init() {
	nodesCmd.AddCommand(getNodeCmd)
	nodesCmd.AddCommand(listNodesCmd)

	// listNodesCmd flags.
	listNodesCmd.Flags().IntSliceVarP(&listNodesCmdFlagValues.Status, "status", "s", nil,
		fmt.Sprintf("Node statuses, options: %s", enumOptions(scheduler_proto.Node_Status_name)))
	listNodesCmd.Flags().Int32VarP(&listNodesCmdFlagValues.Ordering, "order", "o", 0,
		fmt.Sprintf("Ordering, options: %s", enumOptions(scheduler_proto.ListNodesRequest_Ordering_name)))
	listNodesCmd.Flags().Uint32VarP(&listNodesCmdFlagValues.Limit, "limit", "l", 10, "Nodes limit")
	listNodesCmd.Flags().Uint32Var(&listNodesCmdFlagValues.Offset, "offset", 0, "Nodes offset")
	listNodesCmd.Flags().Float32Var(&listNodesCmdFlagValues.CpuAvailable, "cpu", 0,
		"Min CPU available")
	listNodesCmd.Flags().Int32Var(&listNodesCmdFlagValues.CpuClass, "cpu_class", 0,
		fmt.Sprintf("Min CPU class, options: %s", enumOptions(scheduler_proto.CPUClass_name)))
	listNodesCmd.Flags().Uint32Var(&listNodesCmdFlagValues.MemoryAvailable, "memory", 0, "Min Memory available, Mb")
	listNodesCmd.Flags().Uint32Var(&listNodesCmdFlagValues.GpuAvailable, "gpu", 0, "Min GPU cards available")
	listNodesCmd.Flags().Int32Var(&listNodesCmdFlagValues.GpuClass, "gpu_class", 0,
		fmt.Sprintf("Min GPU class, options: %s", enumOptions(scheduler_proto.GPUClass_name)))
	listNodesCmd.Flags().Uint32Var(&listNodesCmdFlagValues.DiskAvailable, "disk", 0, "Min Disk available, Mb")
	listNodesCmd.Flags().Int32Var(&listNodesCmdFlagValues.DiskClass, "disk_class", 0,
		fmt.Sprintf("Disk class, options: %s", enumOptions(scheduler_proto.DiskClass_name)))
	listNodesCmd.Flags().StringArrayVar(&listNodesCmdFlagValues.CpuLabels, "cpu_label", []string{},
		"CPU labels")
	listNodesCmd.Flags().StringArrayVar(&listNodesCmdFlagValues.GpuLabels, "gpu_label", []string{},
		"GPU labels")
	listNodesCmd.Flags().StringArrayVar(&listNodesCmdFlagValues.DiskLabels, "disk_label", []string{},
		"Disk labels")
	listNodesCmd.Flags().StringArrayVar(&listNodesCmdFlagValues.MemoryLabels, "memory_label", []string{},
		"Memory labels")
	listNodesCmd.Flags().StringArrayVar(&listNodesCmdFlagValues.UsernameLabels, "username_label", []string{},
		"Username labels")
	listNodesCmd.Flags().StringArrayVar(&listNodesCmdFlagValues.NameLabels, "name_label", []string{},
		"Node name labels")
	listNodesCmd.Flags().StringArrayVar(&listNodesCmdFlagValues.OtherLabels, "other_label", []string{},
		"Other labels")
	listNodesCmd.Flags().Uint64Var(&listNodesCmdFlagValues.TasksFinished, "tasks_finished", 0,
		"Min finished Tasks on a Node")
	listNodesCmd.Flags().Uint64Var(&listNodesCmdFlagValues.TasksFailed, "tasks_failed", 0,
		"Max failed Tasks on a Node")

	rootCmd.AddCommand(nodesCmd)
}

var getNodeCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get an existing Node",
	Long:    `Get an existing Node by its ID.`,
	Args:    cobra.ExactArgs(1),
	Example: `> scalemate nodes get 42`,
	Run: func(cmd *cobra.Command, args []string) {
		nodeID, err := strconv.Atoi(args[0])
		if err != nil || nodeID <= 0 {
			fmt.Printf("invalid Node ID: %s\n", args[0])
			return
		}
		job, err := scheduler.GetNodeController(
			client.NewSchedulerClient(schedulerServiceAddr),
			uint64(nodeID))
		scheduler.JSONPbView(logger, os.Stdout, job, err)
	},
}

var listNodesCmd = &cobra.Command{
	Use:     "list",
	Short:   "List existing Nodes",
	Long:    `List existing Nodes that satisfy the criteria.`,
	Example: `> scalemate nodes list --memory 2048`,
	Run: func(cmd *cobra.Command, args []string) {
		response, err := scheduler.ListNodesController(
			client.NewSchedulerClient(schedulerServiceAddr),
			&listNodesCmdFlagValues,
		)
		scheduler.JSONPbView(logger, os.Stdout, response, err)
	},
}

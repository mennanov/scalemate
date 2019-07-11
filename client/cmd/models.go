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

// modelsCmd represents the models command
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Get aggregated hardware models data for the Nodes online",
}

func init() {
	modelsCmd.AddCommand(listCpuModelsCmd)
	modelsCmd.AddCommand(listGpuModelsCmd)
	modelsCmd.AddCommand(listDiskModelsCmd)
	modelsCmd.AddCommand(listMemoryModelsCmd)

	rootCmd.AddCommand(modelsCmd)
}

var listCpuModelsCmd = &cobra.Command{
	Use:   "cpu",
	Short: "Get CPU models",
	Long: fmt.Sprintf(
		"Get aggregated CPU hardware models data for the Nodes online. "+
			"Optional argument values to filter by CPU class: %s",
		enumOptions(scheduler_proto.CPUClass_name)),
	Args: cobra.MaximumNArgs(1),
	Example: `Filter by CPU class:
> scalemate models cpu 10
Get all models:
> scalemate models cpu`,
	Run: func(cmd *cobra.Command, args []string) {
		class := 0
		if len(args) > 0 {
			class, err := strconv.Atoi(args[0])
			if _, ok := scheduler_proto.CPUClass_name[int32(class)]; err != nil || class <= 0 || !ok {
				fmt.Printf("invalid CPU class: %s, possible values: %s\n", args[0],
					enumOptions(scheduler_proto.CPUClass_name))
				return
			}
		}
		response, err := scheduler.ListCpuModelsController(
			client.NewSchedulerFrontEndClient(schedulerServiceAddr),
			scheduler_proto.CPUClass(class))
		scheduler.JSONPbView(logger, os.Stdout, response, err)
	},
}

var listGpuModelsCmd = &cobra.Command{
	Use:   "gpu",
	Short: "Get GPU models",
	Long: fmt.Sprintf(
		"Get aggregated GPU hardware models data for the Nodes online. "+
			"Optional argument values to filter by GPU class: %s",
		enumOptions(scheduler_proto.GPUClass_name)),
	Args: cobra.MaximumNArgs(1),
	Example: `Filter by GPU class:
> scalemate models gpu 10
Get all models:
> scalemate models gpu`,
	Run: func(cmd *cobra.Command, args []string) {
		class := 0
		if len(args) > 0 {
			class, err := strconv.Atoi(args[0])
			if _, ok := scheduler_proto.GPUClass_name[int32(class)]; err != nil || class <= 0 || !ok {
				fmt.Printf("invalid GPU class: %s, possible values: %s\n", args[0],
					enumOptions(scheduler_proto.GPUClass_name))
				return
			}
		}
		response, err := scheduler.ListGpuModelsController(
			client.NewSchedulerFrontEndClient(schedulerServiceAddr),
			scheduler_proto.GPUClass(class))
		scheduler.JSONPbView(logger, os.Stdout, response, err)
	},
}

var listDiskModelsCmd = &cobra.Command{
	Use:   "disk",
	Short: "Get disk models",
	Long: fmt.Sprintf(
		"Get aggregated hardware disk models data for the Nodes online. "+
			"Optional argument values to filter by disk class: %s",
		enumOptions(scheduler_proto.DiskClass_name)),
	Args: cobra.MaximumNArgs(1),
	Example: `Filter by disk class:
> scalemate models disk 10
Get all models:
> scalemate models disk`,
	Run: func(cmd *cobra.Command, args []string) {
		class := 0
		if len(args) > 0 {
			class, err := strconv.Atoi(args[0])
			if _, ok := scheduler_proto.DiskClass_name[int32(class)]; err != nil || class <= 0 || !ok {
				fmt.Printf("invalid disk class: %s, possible values: %s\n", args[0],
					enumOptions(scheduler_proto.DiskClass_name))
				return
			}
		}
		response, err := scheduler.ListDiskModelsController(
			client.NewSchedulerFrontEndClient(schedulerServiceAddr),
			scheduler_proto.DiskClass(class))
		scheduler.JSONPbView(logger, os.Stdout, response, err)
	},
}

var listMemoryModelsCmd = &cobra.Command{
	Use:     "memory",
	Short:   "Get memory models",
	Long:    "Get aggregated memory hardware models data for the Nodes online",
	Args:    cobra.MaximumNArgs(0),
	Example: `> scalemate models memory`,
	Run: func(cmd *cobra.Command, args []string) {
		response, err := scheduler.ListMemoryModelsController(client.NewSchedulerFrontEndClient(schedulerServiceAddr))
		scheduler.JSONPbView(logger, os.Stdout, response, err)
	},
}

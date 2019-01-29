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
	"os"
	"strings"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/spf13/cobra"

	"github.com/mennanov/scalemate/client/scheduler"
)

var jobsCreateCmdFlagValues = scheduler.JobsCreateCmdFlags{}

// jobsCmd represents the jobs command
var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Create, list and get jobs",
}

func enumOptions(enum map[int32]string) string {
	var options []string
	for key, value := range enum {
		options = append(options, fmt.Sprintf("%d - %s", key, value))
	}
	return strings.Join(options, ", ")
}

// jobsCreateCmd represents the job creation command
var jobsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Job",
	Long: `Create a new Job entity which will be scheduled immediately.
Once the Job is scheduled a corresponding Task entity is created.`,
	Args: cobra.RangeArgs(1, 2),
	Example: `Run postgres, make it available on localhost:5432 and download it's data to a local folder ./pg_data:
> scalemate jobs create postgres:latest -p 5432:5432 -v ./pg_data:/var/lib/postgresql -d ./pg_data:./pg_data`,
	Run: func(cmd *cobra.Command, args []string) {
		image := args[0]
		var command string
		if len(args) == 2 {
			command = args[1]
		}
		job, err := scheduler.CreateJobController(
			nil, //client.NewAccountsClient(accounts.ServiceAddr),
			nil, //client.NewSchedulerClient(scheduler.ServiceAddr),
			image,
			command,
			&jobsCreateCmdFlagValues)
		scheduler.CreateJobView(logger, os.Stdout, job, err)
	},
}

func init() {
	jobsCmd.AddCommand(jobsCreateCmd)

	rootCmd.AddCommand(jobsCmd)

	// `scalemate jobs create` command flags definitions below.
	jobsCreateCmd.Flags().StringArrayVarP(&jobsCreateCmdFlagValues.Volumes, "volume", "v", []string{},
		"Docker Volumes in a format 'host path relative to the working dir:container path', e.g. "+
			"'./pg_data:/var/lib/postgres/data'")

	jobsCreateCmd.Flags().StringArrayVarP(&jobsCreateCmdFlagValues.DownloadPaths, "download", "d", []string{},
		"Paths to be downloaded from a Node after the Job is finished in a format "+
			"'path relative to the working dir on a remote Node:local path', e.g. './postgres_data:/tmp/pg_data'")

	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.DownloadExclude, "download_exclude", []string{},
		"Exclude these remote paths from being downloaded")

	jobsCreateCmd.Flags().StringArrayVarP(&jobsCreateCmdFlagValues.UploadPaths, "upload", "u", []string{},
		"Paths to be uploaded to a remote Node before container is started in a format "+
			"'local path:path relative to the working dir on a remote Node', e.g. './:./'")

	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.UploadExclude, "upload_exclude", []string{},
		"Exclude these local paths from being uploaded to a remote Node")

	jobsCreateCmd.Flags().StringArrayVarP(&jobsCreateCmdFlagValues.Ports, "port", "p", []string{},
		"Exposed Ports in a format 'localhost port:container port', e.g. '8080:80'")

	jobsCreateCmd.Flags().StringArrayVarP(&jobsCreateCmdFlagValues.EnvVars, "env", "e", []string{},
		"Set environment variables for a container, .e.g 'color=red'")

	jobsCreateCmd.Flags().StringVar(&jobsCreateCmdFlagValues.Entrypoint, "entrypoint", "",
		"Overwrite the default Docker ENTRYPOINT of the image")

	// Resource limit flags.
	jobsCreateCmd.Flags().Float32Var(&jobsCreateCmdFlagValues.CpuLimit, "cpu_limit", 0,
		"CPU limit, specifies how much of the available CPU resources a container can use")
	jobsCreateCmd.Flags().Int32Var(&jobsCreateCmdFlagValues.CpuClass, "cpu_class", 0,
		fmt.Sprintf("CPU class, options: %s", enumOptions(scheduler_proto.CPUClass_name)))

	jobsCreateCmd.Flags().Uint32Var(&jobsCreateCmdFlagValues.GpuLimit, "gpu_limit", 0,
		"GPU limit, number of GPU cards available for a container")
	jobsCreateCmd.Flags().Int32Var(&jobsCreateCmdFlagValues.GpuClass, "gpu_class", 0,
		fmt.Sprintf("GPU class, options: %s", enumOptions(scheduler_proto.GPUClass_name)))

	jobsCreateCmd.Flags().Uint32Var(&jobsCreateCmdFlagValues.DiskLimit, "disk_limit", 0,
		"Disk limit in MB")
	jobsCreateCmd.Flags().Int32Var(&jobsCreateCmdFlagValues.DiskClass, "disk_class", 0,
		fmt.Sprintf("Disk class, options: %s", enumOptions(scheduler_proto.DiskClass_name)))

	jobsCreateCmd.Flags().Int32Var(&jobsCreateCmdFlagValues.RestartPolicy, "restart", 0,
		fmt.Sprintf("Restart policy, options: %s", enumOptions(scheduler_proto.Job_RestartPolicy_name)))

	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.CpuLabels, "cpu_label", []string{},
		"CPU labels")
	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.GpuLabels, "gpu_label", []string{},
		"GPU labels")
	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.DiskLabels, "disk_label", []string{},
		"Disk labels")
	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.MemoryLabels, "memory_label", []string{},
		"Memory labels")
	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.UsernameLabels, "username_label", []string{},
		"Username labels")
	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.NameLabels, "name_label", []string{},
		"Node name labels")
	jobsCreateCmd.Flags().StringArrayVar(&jobsCreateCmdFlagValues.OtherLabels, "other_label", []string{},
		"Other labels")

	jobsCreateCmd.Flags().BoolVar(&jobsCreateCmdFlagValues.IsDaemon, "daemon", false,
		"Run a container immediately: don't wait for an established p2p connection")

}

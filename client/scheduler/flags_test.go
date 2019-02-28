package scheduler_test

import (
	"testing"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mennanov/scalemate/client/scheduler"
)

func TestJobsCreateCmdFlags_ToJobProto_WithAllFlagsCorrect(t *testing.T) {
	for cpuClass := range scheduler_proto.CPUClass_name {
		for gpuClass := range scheduler_proto.GPUClass_name {
			for diskClass := range scheduler_proto.DiskClass_name {
				for restartPolicy := range scheduler_proto.Job_RestartPolicy_name {
					flags := scheduler.JobsCreateCmdFlags{
						Ports:           []string{"80:8000", "443:4443"},
						Volumes:         []string{"./cwd_relative_path:/path_in_container"},
						DownloadPaths:   []string{"cwd_relative_path/sub_path:local_path"},
						DownloadExclude: []string{"_remote_cache"},
						UploadPaths:     []string{"./:./"},
						UploadExclude:   []string{"_cache"},
						EnvVars:         []string{"foo=bar", "bar="},
						Entrypoint:      "/bin/bash",
						CpuLimit:        1.5,
						CpuClass:        cpuClass,
						GpuLimit:        2,
						GpuClass:        gpuClass,
						DiskLimit:       2000,
						DiskClass:       diskClass,
						MemoryLimit:     1000,
						RestartPolicy:   restartPolicy,
						CpuLabels:       []string{"Intel", "AMD"},
						GpuLabels:       []string{"GTX 1060i"},
						DiskLabels:      []string{"Seagate"},
						MemoryLabels:    []string{"DDR3"},
						UsernameLabels:  []string{"username1", "username2"},
						NameLabels:      []string{"node1", "node2"},
						OtherLabels:     []string{"special_promo_label"},
						IsDaemon:        true,
					}
					job, err := flags.ToJobProto();
					require.NoError(t, err)
					assert.NotNil(t, job)
					// Check the values.
					assert.Equal(t, map[uint32]uint32{
						80:  8000,
						443: 4443,
					}, job.RunConfig.Ports)
					assert.Equal(t, map[string]string{
						"./cwd_relative_path": "/path_in_container",
					}, job.RunConfig.Volumes)
					assert.Equal(t, map[string]string{
						"cwd_relative_path/sub_path": "local_path",
					}, job.RunConfig.DownloadPaths)
					assert.Equal(t, flags.DownloadExclude, job.RunConfig.DownloadPathsExclude)
					assert.Equal(t, map[string]string{
						"./": "./",
					}, job.RunConfig.UploadPaths)
					assert.Equal(t, flags.UploadExclude, job.RunConfig.UploadPathsExclude)
					assert.Equal(t, map[string]string{
						"foo": "bar",
						"bar": "",
					}, job.RunConfig.EnvironmentVariables)
					assert.Equal(t, flags.Entrypoint, job.RunConfig.Entrypoint)
					assert.Equal(t, flags.CpuLimit, job.CpuLimit)
					assert.Equal(t, scheduler_proto.CPUClass(flags.CpuClass), job.CpuClass)
					assert.Equal(t, flags.GpuLimit, job.GpuLimit)
					assert.Equal(t, scheduler_proto.GPUClass(flags.GpuClass), job.GpuClass)
					assert.Equal(t, flags.DiskLimit, job.DiskLimit)
					assert.Equal(t, scheduler_proto.DiskClass(flags.DiskClass), job.DiskClass)
					assert.Equal(t, flags.MemoryLimit, job.MemoryLimit)
					assert.Equal(t, scheduler_proto.Job_RestartPolicy(flags.RestartPolicy), job.RestartPolicy)
					assert.Equal(t, flags.CpuLabels, job.CpuLabels)
					assert.Equal(t, flags.GpuLabels, job.GpuLabels)
					assert.Equal(t, flags.DiskLabels, job.DiskLabels)
					assert.Equal(t, flags.MemoryLabels, job.MemoryLabels)
					assert.Equal(t, flags.UsernameLabels, job.UsernameLabels)
					assert.Equal(t, flags.NameLabels, job.NameLabels)
					assert.Equal(t, flags.OtherLabels, job.OtherLabels)
					assert.Equal(t, flags.IsDaemon, job.IsDaemon)
				}
			}
		}
	}
}

func TestJobsCreateCmdFlags_ToJobProto_WithNoFlags(t *testing.T) {
	emptyFlags := &scheduler.JobsCreateCmdFlags{}
	job, err := emptyFlags.ToJobProto()
	require.NoError(t, err)
	require.NotNil(t, job)
}

func TestJobsCreateCmdFlags_ToJobProto_InvalidInput(t *testing.T) {
	for _, flags := range []*scheduler.JobsCreateCmdFlags{
		// Invalid ports.
		{Ports: []string{":"}},
		{Ports: []string{"a:10"}},
		{Ports: []string{"10"}},
		{Ports: []string{"10:"}},
		// TODO: add more test cases for other fields.
		// Invalid volumes.
		{Volumes: []string{"invalid/path/format"}},
		{Volumes: []string{""}},
		{UploadPaths: []string{"invalid/path/format"}},
		{UploadPaths: []string{""}},
		{DownloadPaths: []string{"invalid/path/format"}},
		{DownloadPaths: []string{""}},
		{Ports: []string{"80"}},
		{Ports: []string{"80:-200"}},
		{Ports: []string{"-200:-200"}},
		{Ports: []string{"-200:80"}},
		{Ports: []string{"NaN:80"}},
		{Ports: []string{"80:NaN"}},
		{Ports: []string{""}},
		{EnvVars: []string{"=value"}},
		{EnvVars: []string{"foo"}},
		{EnvVars: []string{""}},
	} {
		job, err := flags.ToJobProto()
		assert.Error(t, err, "failed for flags: %s", flags)
		assert.Nil(t, job)
	}
}

func TestJobsListCmdFlags_ToListJobsRequestProto(t *testing.T) {
	for _, testCase := range []struct {
		flags         *scheduler.JobsListCmdFlags
		expectedProto *scheduler_proto.ListJobsRequest
	}{
		{
			flags: &scheduler.JobsListCmdFlags{
				Status: []int{
					int(scheduler_proto.Job_STATUS_FINISHED),
					int(scheduler_proto.Job_STATUS_NEW),
				},
				Ordering: int32(scheduler_proto.ListJobsRequest_UPDATED_AT_ASC),
				Limit:    100,
				Offset:   200,
			},
			expectedProto: &scheduler_proto.ListJobsRequest{
				Username: "",
				Status: []scheduler_proto.Job_Status{
					scheduler_proto.Job_STATUS_FINISHED,
					scheduler_proto.Job_STATUS_NEW,
				},
				Ordering: scheduler_proto.ListJobsRequest_UPDATED_AT_ASC,
				Limit:    100,
				Offset:   200,
			},
		},
		{
			flags: &scheduler.JobsListCmdFlags{},
			expectedProto: &scheduler_proto.ListJobsRequest{
				Username: "",
				Status:   nil,
				Ordering: scheduler_proto.ListJobsRequest_CREATED_AT_DESC,
				Limit:    0,
				Offset:   0,
			},
		},
	} {
		assert.Equal(t, testCase.expectedProto, testCase.flags.ToListJobsRequestProto())
	}
}

func TestTasksListCmdFlags_ToListTasksRequestProto(t *testing.T) {
	for _, testCase := range []struct {
		flags         *scheduler.TasksListCmdFlags
		expectedProto *scheduler_proto.ListTasksRequest
	}{
		{
			flags: &scheduler.TasksListCmdFlags{
				Status: []int{
					int(scheduler_proto.Task_STATUS_CANCELLED),
					int(scheduler_proto.Task_STATUS_RUNNING),
				},
				Ordering: int32(scheduler_proto.ListTasksRequest_UPDATED_AT_ASC),
				Limit:    100,
				Offset:   200,
			},
			expectedProto: &scheduler_proto.ListTasksRequest{
				Username: "",
				Status: []scheduler_proto.Task_Status{
					scheduler_proto.Task_STATUS_CANCELLED,
					scheduler_proto.Task_STATUS_RUNNING,
				},
				Ordering: scheduler_proto.ListTasksRequest_UPDATED_AT_ASC,
				Limit:    100,
				Offset:   200,
			},
		},
		{
			flags: &scheduler.TasksListCmdFlags{},
			expectedProto: &scheduler_proto.ListTasksRequest{
				Username: "",
				Status:   nil,
				Ordering: scheduler_proto.ListTasksRequest_CREATED_AT_DESC,
				Limit:    0,
				Offset:   0,
			},
		},
	} {
		assert.Equal(t, testCase.expectedProto, testCase.flags.ToListTasksRequestProto())
	}
}

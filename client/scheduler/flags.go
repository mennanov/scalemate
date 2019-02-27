package scheduler

import (
	"strconv"
	"strings"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
)

// parsePathDefs parses a slice of string like "./local_path:/remote_path" and saves to a map of strings.
func parsePathDefs(to map[string]string, paths []string) error {
	for _, pathDef := range paths {
		parts := strings.Split(pathDef, ":")
		if len(parts) != 2 {
			return errors.Errorf("invalid path format: %s", pathDef)
		}
		to[parts[0]] = parts[1]
	}
	return nil
}

// JobsCreateCmdFlags represents a set of flags for the `scalemate jobs create` command.
// Positional arguments are not included.
type JobsCreateCmdFlags struct {
	Ports           []string
	Volumes         []string
	DownloadPaths   []string
	DownloadExclude []string
	UploadPaths     []string
	UploadExclude   []string
	EnvVars         []string
	Entrypoint      string
	CpuLimit        float32
	CpuClass        int32
	GpuLimit        uint32
	GpuClass        int32
	DiskLimit       uint32
	DiskClass       int32
	MemoryLimit     uint32
	RestartPolicy   int32
	CpuLabels       []string
	GpuLabels       []string
	DiskLabels      []string
	MemoryLabels    []string
	UsernameLabels  []string
	NameLabels      []string
	OtherLabels     []string
	IsDaemon        bool
}

// ToJobProto parses the flags and returns a filled in Job_RunConfig struct.
func (f *JobsCreateCmdFlags) ToJobProto() (*scheduler_proto.Job, error) {
	job := &scheduler_proto.Job{
		RunConfig: &scheduler_proto.Job_RunConfig{
			Entrypoint:           f.Entrypoint,
			UploadPathsExclude:   f.UploadExclude,
			DownloadPathsExclude: f.DownloadExclude,
		},
		CpuLimit:       f.CpuLimit,
		CpuClass:       scheduler_proto.CPUClass(f.CpuClass),
		GpuLimit:       f.GpuLimit,
		GpuClass:       scheduler_proto.GPUClass(f.GpuClass),
		DiskLimit:      f.DiskLimit,
		DiskClass:      scheduler_proto.DiskClass(f.DiskClass),
		MemoryLimit:    f.MemoryLimit,
		RestartPolicy:  scheduler_proto.Job_RestartPolicy(f.RestartPolicy),
		CpuLabels:      f.CpuLabels,
		GpuLabels:      f.GpuLabels,
		DiskLabels:     f.DiskLabels,
		MemoryLabels:   f.MemoryLabels,
		UsernameLabels: f.UsernameLabels,
		NameLabels:     f.NameLabels,
		OtherLabels:    f.OtherLabels,
		IsDaemon:       f.IsDaemon,
	}
	// Volumes flag.
	if len(f.Volumes) != 0 {
		job.RunConfig.Volumes = make(map[string]string)
		if err := parsePathDefs(job.RunConfig.Volumes, f.Volumes); err != nil {
			return nil, errors.Wrap(err, "failed to parse volumes flag")
		}
	}
	// Upload flag.
	if len(f.UploadPaths) != 0 {
		job.RunConfig.UploadPaths = make(map[string]string)
		if err := parsePathDefs(job.RunConfig.UploadPaths, f.UploadPaths); err != nil {
			return nil, errors.Wrap(err, "failed to parse upload flag")
		}
	}

	// Download flag.
	if len(f.DownloadPaths) != 0 {
		job.RunConfig.DownloadPaths = make(map[string]string)
		if err := parsePathDefs(job.RunConfig.DownloadPaths, f.DownloadPaths); err != nil {
			return nil, errors.Wrap(err, "failed to parse download flag")
		}
	}
	// Ports flag.
	if len(f.Ports) != 0 {
		job.RunConfig.Ports = make(map[uint32]uint32)
		for _, portDef := range f.Ports {
			parts := strings.Split(portDef, ":")
			if len(parts) != 2 {
				return nil, errors.Errorf("invalid port format: %s", portDef)
			}
			localPort, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, errors.Errorf("invalid local port format: %s", parts[0])
			}
			remotePort, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, errors.Errorf("invalid remote port format: %s", parts[1])
			}
			job.RunConfig.Ports[uint32(localPort)] = uint32(remotePort)
		}
	}
	// Env variables.
	if len(f.EnvVars) != 0 {
		job.RunConfig.EnvironmentVariables = make(map[string]string)
		for _, envVarDef := range f.EnvVars {
			eqIdx := strings.Index(envVarDef, "=")
			if eqIdx == -1 || eqIdx == 0 {
				return nil, errors.Errorf("invalid env variable format: %s", envVarDef)
			}
			job.RunConfig.EnvironmentVariables[envVarDef[:eqIdx]] = envVarDef[eqIdx+1:]
		}
	}
	return job, nil
}

// JobsListCmdFlags represents a set of flags for the `scalemate jobs list` command.
type JobsListCmdFlags struct {
	Status   []int
	Ordering int32
	Limit    uint32
	Offset   uint32
}

// ToListJobsRequestProto creates a new scheduler_proto.ListJobsRequest from the flags.
func (f *JobsListCmdFlags) ToListJobsRequestProto() *scheduler_proto.ListJobsRequest {
	request := &scheduler_proto.ListJobsRequest{
		Ordering: scheduler_proto.ListJobsRequest_Ordering(f.Ordering),
		Limit:    f.Limit,
		Offset:   f.Offset,
	}
	if len(f.Status) != 0 {
		request.Status = make([]scheduler_proto.Job_Status, len(f.Status))
		for i, s := range f.Status {
			request.Status[i] = scheduler_proto.Job_Status(s)
		}
	}

	return request
}

// TasksListCmdFlags represents a set of flags for the `scalemate tasks list` command.
type TasksListCmdFlags struct {
	Status   []int
	Ordering int32
	Limit    uint32
	Offset   uint32
}

// ToListTasksRequestProto creates a new scheduler_proto.ListTasksRequest from the flags.
func (f *TasksListCmdFlags) ToListTasksRequestProto() *scheduler_proto.ListTasksRequest {
	request := &scheduler_proto.ListTasksRequest{
		Ordering: scheduler_proto.ListTasksRequest_Ordering(f.Ordering),
		Limit:    f.Limit,
		Offset:   f.Offset,
	}
	if len(f.Status) != 0 {
		request.Status = make([]scheduler_proto.Task_Status, len(f.Status))
		for i, s := range f.Status {
			request.Status[i] = scheduler_proto.Task_Status(s)
		}
	}

	return request
}

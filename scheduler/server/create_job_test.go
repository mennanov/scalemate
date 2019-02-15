package server_test

import (
	"context"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestCreateJob_ValidMinimalRequest() {
	res, err := s.client.CreateJob(context.Background(), s.newJob())
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.NotEqual(uint64(0), res.Id)

	utils.WaitForMessages(s.amqpRawConsumer, "scheduler.job.created")
}

func (s *ServerTestSuite) TestCreateJob_ValidFullRequest() {
	req := &scheduler_proto.Job{
		Username: s.claimsInjector.Claims.Username,
		RunConfig: &scheduler_proto.Job_RunConfig{
			Image:                "image",
			Command:              "command",
			Ports:                map[uint32]uint32{8080: 8080},
			Volumes:              map[string]string{"./db_data": "/var/lib/postgres"},
			DownloadPaths:        map[string]string{"./db_data": "/home/username/db"},
			DownloadPathsExclude: []string{"_cache"},
			UploadPaths:          map[string]string{"/home/username/db": "./db_data"},
			UploadPathsExclude:   []string{"_cache"},
			EnvironmentVariables: map[string]string{"foo": "bar"},
			Entrypoint:           "entrypoint",
		},
		CpuLimit:       0.5,
		CpuClass:       scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
		MemoryLimit:    1,
		GpuLimit:       2,
		GpuClass:       scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
		DiskLimit:      1,
		DiskClass:      scheduler_proto.DiskClass_DISK_CLASS_SSD,
		RestartPolicy:  scheduler_proto.Job_RESTART_POLICY_ON_FAILURE,
		CpuLabels:      []string{"i7"},
		GpuLabels:      []string{"nVidia"},
		DiskLabels:     []string{"disk label"},
		MemoryLabels:   []string{"memory label"},
		UsernameLabels: []string{"username label"},
		NameLabels:     []string{"name label"},
		OtherLabels:    []string{"other"},
		IsDaemon:       true,
	}
	res, err := s.client.CreateJob(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.NotEqual(uint64(0), res.Id)
	utils.WaitForMessages(s.amqpRawConsumer, "scheduler.job.created")
}

func (s *ServerTestSuite) TestCreateJob_InvalidRequests() {
	for _, testCase := range []struct {
		job  *scheduler_proto.Job
		code codes.Code
	}{
		// Username does not match the one in auth claims.
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.Username = "invalid_username"
			}),
			codes.PermissionDenied,
		},
		// Read-only fields are filled-in.
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.Id = 1
			}),
			codes.InvalidArgument,
		},
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.Status = scheduler_proto.Job_STATUS_PENDING
			}),
			codes.InvalidArgument,
		},
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.CreatedAt = &timestamp.Timestamp{}
			}),
			codes.InvalidArgument,
		},
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.UpdatedAt = &timestamp.Timestamp{}
			}),
			codes.InvalidArgument,
		},
		// Invalid network ports.
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.RunConfig = s.newRunConfig(func(config *scheduler_proto.Job_RunConfig) {
					config.Ports = map[uint32]uint32{65536: 10}
				})
			}),
			codes.InvalidArgument,
		},
		// Invalid volume paths.
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.RunConfig = s.newRunConfig(func(config *scheduler_proto.Job_RunConfig) {
					config.Volumes = map[string]string{"/abs/host": "/abs/container"}
				})
			}),
			codes.InvalidArgument,
		},
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.RunConfig = s.newRunConfig(func(config *scheduler_proto.Job_RunConfig) {
					config.Volumes = map[string]string{"../host": "/abs/container"}
				})
			}),
			codes.InvalidArgument,
		},
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.RunConfig = s.newRunConfig(func(config *scheduler_proto.Job_RunConfig) {
					config.Volumes = map[string]string{"./../host": "/abs/container"}
				})
			}),
			codes.InvalidArgument,
		},
		{
			s.newJob(func(job *scheduler_proto.Job) {
				job.RunConfig = s.newRunConfig(func(config *scheduler_proto.Job_RunConfig) {
					config.Volumes = map[string]string{"folder/../../etc/passwd": "/abs/container"}
				})
			}),
			codes.InvalidArgument,
		},
	} {
		res, err := s.client.CreateJob(context.Background(), testCase.job)
		s.assertGRPCError(err, testCase.code)
		s.Nil(res)
	}
}

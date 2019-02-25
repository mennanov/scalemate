package scheduler_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/client/scheduler"
)

func createTestingJob(createdAt, updatedAt *timestamp.Timestamp) *scheduler_proto.Job {
	return &scheduler_proto.Job{
		Id:       1,
		Username: "username",
		Status:   scheduler_proto.Job_STATUS_PENDING,
		RunConfig: &scheduler_proto.Job_RunConfig{
			Image:                "image",
			Command:              "command",
			Ports:                map[uint32]uint32{80: 80},
			Volumes:              map[string]string{"a/b": "/etc/a/b"},
			DownloadPaths:        map[string]string{"./": "./"},
			DownloadPathsExclude: []string{"_cache"},
			UploadPaths:          map[string]string{"./": "./"},
			UploadPathsExclude:   []string{"_local_cache"},
			EnvironmentVariables: map[string]string{"foo": "bar"},
			Entrypoint:           "/custom/entrypoint",
		},
		CpuLimit:      1.5,
		CpuClass:      scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
		MemoryLimit:   1000,
		GpuLimit:      2,
		GpuClass:      scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
		DiskLimit:     2000,
		DiskClass:     scheduler_proto.DiskClass_DISK_CLASS_HDD,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		RestartPolicy: scheduler_proto.Job_RESTART_POLICY_ON_FAILURE,
		IsDaemon:      false,
	}
}

func TestCreateJobView(t *testing.T) {
	t.Run("WithError", func(t *testing.T) {
		for _, err := range []error{
			status.Error(codes.Unauthenticated, "unknown JWT claims type"),
			status.Error(codes.PermissionDenied, "wrong username"),
			status.Error(codes.InvalidArgument, "validation error"),
			status.Error(codes.Internal, "internal error"),
			errors.New("unknown error"),
		} {
			logger, hook := test.NewNullLogger()

			output := new(bytes.Buffer)
			scheduler.CreateJobView(logger, output, nil, err)
			// Verify that the error is logged with an appropriate level.
			assert.Equal(t, 1, len(hook.Entries))
			assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
			// Check that the output is empty.
			assert.Equal(t, 0, output.Len())
		}
	})

	t.Run("JobJSONEncoded", func(t *testing.T) {
		now, err := ptypes.TimestampProto(time.Now())
		require.NoError(t, err)
		job := createTestingJob(now, now)
		logger, hook := test.NewNullLogger()

		output := new(bytes.Buffer)
		scheduler.CreateJobView(logger, output, job, err)
		// Verify that there are no errors logged (only info messages).
		assert.Equal(t, 1, len(hook.Entries))
		assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
		// Check that the output is a valid JSON.
		jsonDecodedJob := &scheduler_proto.Job{}
		require.NoError(t, jsonpb.Unmarshal(output, jsonDecodedJob))
		assert.Equal(t, job, jsonDecodedJob)
	})
}

func TestGetJobView(t *testing.T) {
	t.Run("WithError", func(t *testing.T) {
		for _, err := range []error{
			status.Error(codes.Unauthenticated, "unknown JWT claims type"),
			status.Error(codes.PermissionDenied, "wrong username"),
			status.Error(codes.InvalidArgument, "validation error"),
			status.Error(codes.Internal, "internal error"),
			errors.New("unknown error"),
		} {
			logger, hook := test.NewNullLogger()

			output := new(bytes.Buffer)
			scheduler.GetJobView(logger, output, nil, err)
			// Verify that the error is logged with an appropriate level.
			assert.Equal(t, 1, len(hook.Entries))
			assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
			// Check that the output is empty.
			assert.Equal(t, 0, output.Len())
		}
	})

	t.Run("JobJSONEncoded", func(t *testing.T) {
		now, err := ptypes.TimestampProto(time.Now())
		require.NoError(t, err)
		job := createTestingJob(now, now)
		logger, hook := test.NewNullLogger()

		output := new(bytes.Buffer)
		scheduler.GetJobView(logger, output, job, err)
		// Verify that there are no logged entries.
		assert.Equal(t, 0, len(hook.Entries))
		// Check that the output is a valid JSON.
		jsonDecodedJob := &scheduler_proto.Job{}
		require.NoError(t, jsonpb.Unmarshal(output, jsonDecodedJob))
		assert.Equal(t, job, jsonDecodedJob)
	})
}

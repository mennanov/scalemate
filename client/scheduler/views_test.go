package scheduler_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mennanov/scalemate/client/scheduler"
	"github.com/mennanov/scalemate/shared/utils"
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
			RestartPolicy:        scheduler_proto.Job_RunConfig_RESTART_POLICY_ON_FAILURE,
		},
		CpuLimit:         1.5,
		CpuClass:         scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
		MemoryLimit:      1000,
		GpuLimit:         2,
		GpuClass:         scheduler_proto.GPUClass_GPU_CLASS_ADVANCED,
		DiskLimit:        2000,
		DiskClass:        scheduler_proto.DiskClass_DISK_CLASS_HDD,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		ReschedulePolicy: scheduler_proto.Job_RESCHEDULE_POLICY_ON_NODE_FAILURE,
		IsDaemon:         false,
	}
}

func TestJsonPbView(t *testing.T) {
	t.Run("WithError", func(t *testing.T) {
		for _, err := range utils.GetAllErrors() {
			logger, hook := test.NewNullLogger()

			output := new(bytes.Buffer)
			scheduler.JSONPbView(logger, output, nil, err)
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
		scheduler.JSONPbView(logger, output, job, err)
		// Verify that there are no logged entries.
		assert.Equal(t, 0, len(hook.Entries))
		// Check that the output is a valid JSON.
		jsonDecodedJob := &scheduler_proto.Job{}
		require.NoError(t, jsonpb.Unmarshal(output, jsonDecodedJob))
		assert.Equal(t, job, jsonDecodedJob)
	})

	t.Run("MarshalFails", func(t *testing.T) {
		logger, hook := test.NewNullLogger()
		output := new(bytes.Buffer)
		scheduler.JSONPbView(logger, output, nil, nil)
		// Verify that the error is logged with an appropriate level.
		require.Equal(t, 1, len(hook.Entries))
		assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
	})
}

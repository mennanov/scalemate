package scheduler_test

import (
	"context"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/client/scheduler"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/testutils"
)

func TestCreateJobController(t *testing.T) {
	t.Run("WithNoFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		username := "username"
		image := "image"
		jobRequestExpected := &scheduler_proto.Job{
			Username:  username,
			RunConfig: &scheduler_proto.Job_RunConfig{Image: image},
		}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CreateJob(ctx, jobRequestExpected, gomock.Any()).Return(&scheduler_proto.Job{}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			image, "", &scheduler.CreateJobCmdFlags{})
		require.NoError(t, err)
		assert.NotNil(t, job)
	})

	t.Run("WithFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		username := "username"
		image := "image"
		command := "command"
		entrypoint := "entrypoint"
		jobRequestExpected := &scheduler_proto.Job{
			Username: username,
			RunConfig: &scheduler_proto.Job_RunConfig{
				Image:      image,
				Command:    command,
				Ports:      map[uint32]uint32{8080: 8080},
				Volumes:    map[string]string{"./path": "/path"},
				Entrypoint: entrypoint,
			},
		}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CreateJob(ctx, jobRequestExpected, gomock.Any()).Return(&scheduler_proto.Job{}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			image, command, &scheduler.CreateJobCmdFlags{
				Ports:      []string{"8080:8080"},
				Volumes:    []string{"./path:/path"},
				Entrypoint: entrypoint,
			})
		require.NoError(t, err)
		assert.NotNil(t, job)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			"image", "", &scheduler.CreateJobCmdFlags{})
		assert.Error(t, err)
	})

	t.Run("InvalidAuthTokens", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		authTokens := &accounts_proto.AuthTokens{
			AccessToken:  "invalid jwt access token",
			RefreshToken: "invalid jwt refresh token",
		}
		err := auth.SaveTokens(authTokens)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, auth.DeleteTokens())
		}()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err = scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			"", "", &scheduler.CreateJobCmdFlags{})
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		username := "username"
		jobRequestExpected := &scheduler_proto.Job{
			Username:  username,
			RunConfig: &scheduler_proto.Job_RunConfig{Image: "image"},
		}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CreateJob(ctx, jobRequestExpected, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			"image", "", &scheduler.CreateJobCmdFlags{})
		require.Error(t, err)
		assert.Nil(t, job)
	})
}

func TestGetJobController(t *testing.T) {
	t.Run("JobSuccessfullyReturned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		jobId := uint64(42)
		jobLookupRequestExpected := &scheduler_proto.JobLookupRequest{JobId: jobId}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().GetJob(ctx, jobLookupRequestExpected, gomock.Any()).
			Return(&scheduler_proto.Job{Id: jobId}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_user")
		defer deleteTokens()

		job, err := scheduler.GetJobController(NewMockAccountsClient(ctrl), schedulerClient, jobId)
		require.NoError(t, err)
		assert.NotNil(t, job)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.GetJobController(NewMockAccountsClient(ctrl), schedulerClient, uint64(42))
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		jobId := uint64(42)
		jobLookupRequestExpected := &scheduler_proto.JobLookupRequest{JobId: jobId}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().GetJob(ctx, jobLookupRequestExpected, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_username")
		defer deleteTokens()

		job, err := scheduler.GetJobController(NewMockAccountsClient(ctrl), schedulerClient, jobId)
		require.Error(t, err)
		assert.Nil(t, job)
	})
}

func TestListJobsController(t *testing.T) {
	t.Run("WithNoFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		flags := &scheduler.ListJobsCmdFlags{}
		username := "test_username"
		listJobsRequestExpected := &scheduler_proto.ListJobsRequest{Username: username}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListJobs(ctx, listJobsRequestExpected, gomock.Any()).
			Return(&scheduler_proto.ListJobsResponse{TotalCount: 0}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListJobsController(NewMockAccountsClient(ctrl), schedulerClient, flags)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("WithAllFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		flags := &scheduler.ListJobsCmdFlags{
			Status:   []int{int(scheduler_proto.Job_STATUS_FINISHED)},
			Ordering: int32(scheduler_proto.ListJobsRequest_CREATED_AT_ASC),
			Limit:    150,
			Offset:   50,
		}
		username := "test_username"
		listJobsRequestExpected := &scheduler_proto.ListJobsRequest{
			Username: username,
			Status:   []scheduler_proto.Job_Status{scheduler_proto.Job_STATUS_FINISHED},
			Ordering: scheduler_proto.ListJobsRequest_CREATED_AT_ASC,
			Limit:    150,
			Offset:   50,
		}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListJobs(ctx, listJobsRequestExpected, gomock.Any()).
			Return(&scheduler_proto.ListJobsResponse{TotalCount: 0}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListJobsController(NewMockAccountsClient(ctrl), schedulerClient, flags)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.ListJobsController(NewMockAccountsClient(ctrl), schedulerClient, nil)
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		username := "test_user"

		listJobsRequestExpected := &scheduler_proto.ListJobsRequest{Username: username}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListJobs(ctx, listJobsRequestExpected, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListJobsController(NewMockAccountsClient(ctrl), schedulerClient,
			&scheduler.ListJobsCmdFlags{})
		require.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestCancelJobController(t *testing.T) {
	t.Run("JobSuccessfullyCancelled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		jobId := uint64(42)
		jobLookupRequestExpected := &scheduler_proto.JobLookupRequest{JobId: jobId}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CancelJob(ctx, jobLookupRequestExpected, gomock.Any()).
			Return(&scheduler_proto.Job{Id: jobId}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_user")
		defer deleteTokens()

		job, err := scheduler.CancelJobController(NewMockAccountsClient(ctrl), schedulerClient, jobId)
		require.NoError(t, err)
		assert.NotNil(t, job)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.CancelJobController(NewMockAccountsClient(ctrl), schedulerClient, uint64(42))
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		jobId := uint64(42)
		jobLookupRequestExpected := &scheduler_proto.JobLookupRequest{JobId: jobId}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CancelJob(ctx, jobLookupRequestExpected, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_user")
		defer deleteTokens()

		job, err := scheduler.CancelJobController(NewMockAccountsClient(ctrl), schedulerClient, jobId)
		require.Error(t, err)
		assert.Nil(t, job)
	})
}

func TestGetTaskController(t *testing.T) {
	t.Run("TaskSuccessfullyReturned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		taskID := uint64(42)
		taskLookupRequestExpected := &scheduler_proto.TaskLookupRequest{TaskId: taskID}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().GetTask(ctx, taskLookupRequestExpected, gomock.Any()).
			Return(&scheduler_proto.Task{Id: taskID}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_user")
		defer deleteTokens()

		task, err := scheduler.GetTaskController(NewMockAccountsClient(ctrl), schedulerClient, taskID)
		require.NoError(t, err)
		assert.NotNil(t, task)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.GetTaskController(NewMockAccountsClient(ctrl), schedulerClient, uint64(42))
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		taskID := uint64(42)
		taskLookupRequestExpected := &scheduler_proto.TaskLookupRequest{TaskId: taskID}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().GetTask(ctx, taskLookupRequestExpected, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_username")
		defer deleteTokens()

		task, err := scheduler.GetTaskController(NewMockAccountsClient(ctrl), schedulerClient, taskID)
		require.Error(t, err)
		assert.Nil(t, task)
	})
}

func TestListTasksController(t *testing.T) {
	t.Run("WithNoFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		flags := &scheduler.ListTasksCmdFlags{}
		jobIDs := []uint64{1, 2}
		username := "test_username"
		listTasksRequestExpected := &scheduler_proto.ListTasksRequest{Username: username, JobId: jobIDs}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListTasks(ctx, listTasksRequestExpected, gomock.Any()).
			Return(&scheduler_proto.ListTasksResponse{TotalCount: 0}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListTasksController(NewMockAccountsClient(ctrl), schedulerClient, jobIDs, flags)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("WithAllFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		jobIDs := []uint64{1, 2}
		flags := &scheduler.ListTasksCmdFlags{
			Status:   []int{int(scheduler_proto.Task_STATUS_RUNNING)},
			Ordering: int32(scheduler_proto.ListTasksRequest_UPDATED_AT_ASC),
			Limit:    150,
			Offset:   50,
		}
		username := "test_username"
		listJobsRequestExpected := &scheduler_proto.ListTasksRequest{
			Username: username,
			JobId:    jobIDs,
			Status:   []scheduler_proto.Task_Status{scheduler_proto.Task_STATUS_RUNNING},
			Ordering: scheduler_proto.ListTasksRequest_UPDATED_AT_ASC,
			Limit:    150,
			Offset:   50,
		}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListTasks(ctx, listJobsRequestExpected, gomock.Any()).
			Return(&scheduler_proto.ListTasksResponse{TotalCount: 0}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListTasksController(NewMockAccountsClient(ctrl), schedulerClient, jobIDs, flags)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.ListTasksController(NewMockAccountsClient(ctrl), schedulerClient, nil, nil)
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()
		username := "test_user"
		jobIDs := []uint64{1, 2}
		listTasksRequestExpected := &scheduler_proto.ListTasksRequest{Username: username, JobId: jobIDs}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListTasks(ctx, listTasksRequestExpected, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListTasksController(NewMockAccountsClient(ctrl), schedulerClient,
			jobIDs, &scheduler.ListTasksCmdFlags{})
		require.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestCancelTaskController(t *testing.T) {
	t.Run("TaskSuccessfullyCancelled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		taskID := uint64(42)
		taskLookupRequestExpected := &scheduler_proto.TaskLookupRequest{TaskId: taskID}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CancelTask(ctx, taskLookupRequestExpected, gomock.Any()).
			Return(&scheduler_proto.Task{Id: taskID}, nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_user")
		defer deleteTokens()

		task, err := scheduler.CancelTaskController(NewMockAccountsClient(ctrl), schedulerClient, taskID)
		require.NoError(t, err)
		assert.NotNil(t, task)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.CancelTaskController(NewMockAccountsClient(ctrl), schedulerClient, uint64(42))
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		taskID := uint64(42)
		taskLookupRequestExpected := &scheduler_proto.TaskLookupRequest{TaskId: taskID}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CancelTask(ctx, taskLookupRequestExpected, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_username")
		defer deleteTokens()

		task, err := scheduler.CancelTaskController(NewMockAccountsClient(ctrl), schedulerClient, taskID)
		require.Error(t, err)
		assert.Nil(t, task)
	})
}

type tasksStreamingFakeClient struct {
	index int
	tasks []*scheduler_proto.Task
}

func (c *tasksStreamingFakeClient) Recv() (*scheduler_proto.Task, error) {
	if c.index < len(c.tasks) {
		c.index++
		return c.tasks[c.index-1], nil
	}
	return nil, io.EOF
}

func (c *tasksStreamingFakeClient) Header() (metadata.MD, error) {
	return nil, nil
}

func (c *tasksStreamingFakeClient) Trailer() metadata.MD {
	return nil
}

func (c *tasksStreamingFakeClient) CloseSend() error {
	return nil
}

func (c *tasksStreamingFakeClient) Context() context.Context {
	return nil
}

func (c *tasksStreamingFakeClient) SendMsg(m interface{}) error {
	return nil
}

func (c *tasksStreamingFakeClient) RecvMsg(m interface{}) error {
	return nil
}

// Compile time interface check.
var _ scheduler_proto.Scheduler_IterateTasksClient = new(tasksStreamingFakeClient)

func TestIterateTasksController(t *testing.T) {
	t.Run("TasksSuccessfullyReceived", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		jobID := uint64(42)
		iterateTasksRequest := &scheduler_proto.IterateTasksRequest{JobId: jobID}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().IterateTasks(ctx, iterateTasksRequest, gomock.Any()).
			Return(new(tasksStreamingFakeClient), nil)

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_user")
		defer deleteTokens()

		client, err := scheduler.IterateTasksController(NewMockAccountsClient(ctrl), schedulerClient, jobID, false)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("NotLoggedIn", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		schedulerClient := NewMockSchedulerClient(ctrl)

		_, err := scheduler.IterateTasksController(NewMockAccountsClient(ctrl), schedulerClient, uint64(42), true)
		assert.Error(t, err)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		jobID := uint64(42)
		iterateTasksRequest := &scheduler_proto.IterateTasksRequest{JobId: jobID}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().IterateTasks(ctx, iterateTasksRequest, gomock.Any()).
			Return(nil, status.Error(codes.PermissionDenied, "permission denied"))

		deleteTokens := testutils.CreateAndSaveTestingTokens(t, "test_username")
		defer deleteTokens()

		task, err := scheduler.IterateTasksController(NewMockAccountsClient(ctrl), schedulerClient, jobID, false)
		require.Error(t, err)
		assert.Nil(t, task)
	})
}

func TestGetNodeController(t *testing.T) {
	t.Run("Node successfully returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		nodeID := uint64(42)
		nodeLookupRequestExpected := &scheduler_proto.NodeLookupRequest{NodeId: nodeID}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().GetNode(ctx, nodeLookupRequestExpected, gomock.Any()).
			Return(&scheduler_proto.Node{Id: nodeID}, nil)

		response, err := scheduler.GetNodeController(schedulerClient, nodeID)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})
}

func TestListCpuModelsController(t *testing.T) {
	t.Run("CPU models returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		class := scheduler_proto.CPUClass_CPU_CLASS_INTERMEDIATE
		requestExpected := &scheduler_proto.ListCpuModelsRequest{CpuClass: class}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListCpuModels(ctx, requestExpected, gomock.Any()).
			Return(&scheduler_proto.ListCpuModelsResponse{}, nil)

		response, err := scheduler.ListCpuModelsController(schedulerClient, class)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})
}

func TestListGpuModelsController(t *testing.T) {
	t.Run("GPU models returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		class := scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE
		requestExpected := &scheduler_proto.ListGpuModelsRequest{GpuClass: class}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListGpuModels(ctx, requestExpected, gomock.Any()).
			Return(&scheduler_proto.ListGpuModelsResponse{}, nil)

		response, err := scheduler.ListGpuModelsController(schedulerClient, class)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})
}

func TestListDiskModelsController(t *testing.T) {
	t.Run("Disk models returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		class := scheduler_proto.DiskClass_DISK_CLASS_SSD
		requestExpected := &scheduler_proto.ListDiskModelsRequest{DiskClass: class}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListDiskModels(ctx, requestExpected, gomock.Any()).
			Return(&scheduler_proto.ListDiskModelsResponse{}, nil)

		response, err := scheduler.ListDiskModelsController(schedulerClient, class)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})
}

func TestListMemoryModelsController(t *testing.T) {
	t.Run("Memory models returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListMemoryModels(ctx, &empty.Empty{}, gomock.Any()).
			Return(&scheduler_proto.ListMemoryModelsResponse{}, nil)

		response, err := scheduler.ListMemoryModelsController(schedulerClient)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})
}

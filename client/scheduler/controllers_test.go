package scheduler_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/client/scheduler"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			image, "", &scheduler.JobsCreateCmdFlags{})
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			image, command, &scheduler.JobsCreateCmdFlags{
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
			"image", "", &scheduler.JobsCreateCmdFlags{})
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
			"", "", &scheduler.JobsCreateCmdFlags{})
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			"image", "", &scheduler.JobsCreateCmdFlags{})
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_user")
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_username")
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

		flags := &scheduler.JobsListCmdFlags{}
		username := "test_username"
		listJobsRequestExpected := &scheduler_proto.ListJobsRequest{Username: username}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListJobs(ctx, listJobsRequestExpected, gomock.Any()).
			Return(&scheduler_proto.ListJobsResponse{TotalCount: 0}, nil)

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListJobsController(NewMockAccountsClient(ctrl), schedulerClient, flags)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("WithAllFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		flags := &scheduler.JobsListCmdFlags{
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListJobsController(NewMockAccountsClient(ctrl), schedulerClient,
			&scheduler.JobsListCmdFlags{})
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_user")
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_user")
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_user")
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_username")
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

		flags := &scheduler.TasksListCmdFlags{}
		jobIDs := []uint64{1, 2}
		username := "test_username"
		listTasksRequestExpected := &scheduler_proto.ListTasksRequest{Username: username, JobId: jobIDs}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().ListTasks(ctx, listTasksRequestExpected, gomock.Any()).
			Return(&scheduler_proto.ListTasksResponse{TotalCount: 0}, nil)

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
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
		flags := &scheduler.TasksListCmdFlags{
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		response, err := scheduler.ListTasksController(NewMockAccountsClient(ctrl), schedulerClient,
			jobIDs, &scheduler.TasksListCmdFlags{})
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_user")
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

		deleteTokens := utils.CreateAndSaveTestingTokens(t, "test_username")
		defer deleteTokens()

		task, err := scheduler.CancelTaskController(NewMockAccountsClient(ctrl), schedulerClient, taskID)
		require.Error(t, err)
		assert.Nil(t, task)
	})
}
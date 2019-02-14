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

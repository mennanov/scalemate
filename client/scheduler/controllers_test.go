package scheduler_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		jobRequestExpected := &scheduler_proto.Job{
			Username:  username,
			RunConfig: &scheduler_proto.Job_RunConfig{Image: "image"},
		}
		jobResponse := &scheduler_proto.Job{
			Id:        1,
			Username:  username,
			RunConfig: &scheduler_proto.Job_RunConfig{Image: "image"},
		}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CreateJob(ctx, jobRequestExpected, gomock.Any()).Return(jobResponse, nil)

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			"image", "", &scheduler.JobsCreateCmdFlags{})
		require.NoError(t, err)
		assert.NotNil(t, job)
	})

	t.Run("WithFlags", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx := context.Background()

		username := "username"
		jobRequestExpected := &scheduler_proto.Job{
			Username:  username,
			RunConfig: &scheduler_proto.Job_RunConfig{Image: "image"},
		}
		jobResponse := &scheduler_proto.Job{
			Id:        1,
			Username:  username,
			RunConfig: &scheduler_proto.Job_RunConfig{Image: "image"},
		}

		schedulerClient := NewMockSchedulerClient(ctrl)
		schedulerClient.EXPECT().CreateJob(ctx, jobRequestExpected, gomock.Any()).Return(jobResponse, nil)

		deleteTokens := utils.CreateAndSaveTestingTokens(t, username)
		defer deleteTokens()

		job, err := scheduler.CreateJobController(NewMockAccountsClient(ctrl), schedulerClient,
			"image", "", &scheduler.JobsCreateCmdFlags{})
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
			"image", "", &scheduler.JobsCreateCmdFlags{})
		assert.Error(t, err)
	})
}

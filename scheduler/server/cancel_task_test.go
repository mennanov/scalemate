package server_test

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestCancelTask() {
	node := &models.Node{
		Username: "node_username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	for _, testCase := range []struct {
		job               *models.Container
		task              *models.Task
		claims            *auth.Claims
		expectedErrorCode codes.Code
	}{
		{
			job: &models.Container{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
			},
			task: &models.Task{
				Status: utils.Enum(scheduler_proto.Task_STATUS_NEW),
			},
			claims:            &auth.Claims{Username: "test_username"},
			expectedErrorCode: 0,
		},
		{
			job: &models.Container{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_FINISHED),
			},
			task: &models.Task{
				Status: utils.Enum(scheduler_proto.Task_STATUS_CANCELLED),
			},
			claims:            &auth.Claims{Username: "test_username"},
			expectedErrorCode: codes.FailedPrecondition,
		},
		{
			job: &models.Container{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
			},
			task: &models.Task{
				Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
			},
			claims:            &auth.Claims{Username: "test_username", Role: accounts_proto.User_ADMIN},
			expectedErrorCode: 0,
		},
		{
			job: &models.Container{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_FINISHED),
			},
			task: &models.Task{
				Status: utils.Enum(scheduler_proto.Task_STATUS_CANCELLED),
			},
			claims:            &auth.Claims{Username: "test_username", Role: accounts_proto.User_ADMIN},
			expectedErrorCode: codes.FailedPrecondition,
		},
		{
			job: &models.Container{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
			},
			task: &models.Task{
				Status: utils.Enum(scheduler_proto.Task_STATUS_RUNNING),
			},
			claims:            &auth.Claims{Username: "different_username"},
			expectedErrorCode: codes.PermissionDenied,
		},
	} {
		_, err := testCase.job.Create(s.db)
		s.Require().NoError(err)

		testCase.task.JobID = testCase.job.ID
		testCase.task.NodeID = node.ID
		_, err = testCase.task.Create(s.db)
		s.Require().NoError(err)

		restoreClaims := s.claimsInjector.SetClaims(testCase.claims)

		ctx := context.Background()
		taskProto, err := s.client.CancelTask(ctx, &scheduler_proto.TaskLookupRequest{TaskId: testCase.task.ID})
		restoreClaims()

		if testCase.expectedErrorCode != 0 {
			s.assertGRPCError(err, testCase.expectedErrorCode)
			s.Nil(taskProto)
			continue
		}
		s.NoError(err)
		s.Equal(scheduler_proto.Task_STATUS_CANCELLED, taskProto.GetStatus())
		events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.task.updated")
	}
}

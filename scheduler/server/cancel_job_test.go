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

func (s *ServerTestSuite) TestCancelJob() {
	node := &models.Node{
		Username: "node_username",
		Name:     "node_name",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	for _, testCase := range []struct {
		job               *models.Job
		tasks             []*models.Task
		claims            *auth.Claims
		expectedErrorCode codes.Code
	}{
		{
			job: &models.Job{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
			},
			tasks: []*models.Task{
				{
					Status: utils.Enum(scheduler_proto.Task_STATUS_NEW),
				},
				{
					Status: utils.Enum(scheduler_proto.Task_STATUS_FINISHED),
				},
				{
					Status: utils.Enum(scheduler_proto.Task_STATUS_FAILED),
				},
			},
			claims:            &auth.Claims{Username: "test_username"},
			expectedErrorCode: 0,
		},
		{
			job: &models.Job{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_CANCELLED),
			},
			claims:            &auth.Claims{Username: "test_username"},
			expectedErrorCode: codes.FailedPrecondition,
		},
		{
			job: &models.Job{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
			},
			claims:            &auth.Claims{Username: "test_username", Role: accounts_proto.User_ADMIN},
			expectedErrorCode: 0,
		},
		{
			job: &models.Job{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_FINISHED),
			},
			claims:            &auth.Claims{Username: "test_username", Role: accounts_proto.User_ADMIN},
			expectedErrorCode: codes.FailedPrecondition,
		},
		{
			job: &models.Job{
				Username: "test_username",
				Status:   utils.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
			},
			claims:            &auth.Claims{Username: "different_username"},
			expectedErrorCode: codes.PermissionDenied,
		},
	} {
		_, err := testCase.job.Create(s.db)
		s.Require().NoError(err)

		for _, task := range testCase.tasks {
			task.JobID = testCase.job.ID
			task.NodeID = node.ID
			_, err := task.Create(s.db)
			s.Require().NoError(err)
		}

		restoreClaims := s.claimsInjector.SetClaims(testCase.claims)

		ctx := context.Background()
		jobProto, err := s.client.CancelJob(ctx, &scheduler_proto.JobLookupRequest{JobId: testCase.job.ID})
		restoreClaims()

		if testCase.expectedErrorCode != 0 {
			s.assertGRPCError(err, testCase.expectedErrorCode)
			s.Nil(jobProto)
			continue
		}
		s.NoError(err)
		s.Equal(scheduler_proto.Job_STATUS_CANCELLED, jobProto.GetStatus())
		events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.job.updated")

		for _, task := range testCase.tasks {
			if task.IsTerminated() {
				originalStatus := task.Status
				s.Require().NoError(task.LoadFromDB(s.db))
				// Verify that it's status has not changed.
				s.Equal(originalStatus, task.Status, task.String())
			} else {
				s.Require().NoError(task.LoadFromDB(s.db))
				// Verify that the Task is cancelled.
				s.Equal(utils.Enum(scheduler_proto.Task_STATUS_CANCELLED), task.Status)
				events.WaitForMessages(s.amqpRawConsumer, nil, "scheduler.task.updated")
			}
		}
	}
}

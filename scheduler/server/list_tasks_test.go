package server_test

import (
	"context"
	"testing"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
)

func (s *ServerTestSuite) TestListTasks() {
	node := &models.Node{
		Username: "node_username",
	}
	_, err := node.Create(s.db)
	s.Require().NoError(err)

	jobs := []*models.Job{
		{
			Username: "username1",
			Status:   models.Enum(scheduler_proto.Job_STATUS_PENDING),
		},
		{
			Username: "username1",
			Status:   models.Enum(scheduler_proto.Job_STATUS_FINISHED),
		},
		{
			Username: "username2",
		},
	}

	for _, job := range jobs {
		_, err := job.Create(s.db)
		s.Require().NoError(err)
	}

	tasks := []*models.Task{
		{
			Status: models.Enum(scheduler_proto.Task_STATUS_NEW),
			NodeID: node.ID,
			JobID:  jobs[0].ID,
		},
		{
			Status: models.Enum(scheduler_proto.Task_STATUS_FINISHED),
			NodeID: node.ID,
			JobID:  jobs[1].ID,
		},
		{
			Status: models.Enum(scheduler_proto.Task_STATUS_NEW),
			NodeID: node.ID,
			JobID:  jobs[2].ID,
		},
	}

	for _, task := range tasks {
		_, err := task.Create(s.db)
		s.Require().NoError(err)
	}

	ctx := context.Background()

	s.T().Run("returns owned Tasks", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: jobs[0].Username})
		defer restoreClaims()

		req := &scheduler_proto.ListTasksRequest{
			Username: jobs[0].Username,
		}

		res, err := s.client.ListTasks(ctx, req)
		s.Require().NoError(err)
		s.Equal(uint32(2), res.TotalCount)
		s.Equal(tasks[1].ID, res.Tasks[0].Id)
		s.Equal(tasks[0].ID, res.Tasks[1].Id)
	})

	s.T().Run("returns all Tasks for admin", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(
			&auth.Claims{Username: "admin", Role: accounts_proto.User_ADMIN})
		defer restoreClaims()

		req := &scheduler_proto.ListTasksRequest{
			Username: jobs[1].Username,
		}

		res, err := s.client.ListTasks(ctx, req)
		s.Require().NoError(err)
		s.Equal(uint32(2), res.TotalCount)
		s.Equal(tasks[1].ID, res.Tasks[0].Id)
		s.Equal(tasks[0].ID, res.Tasks[1].Id)
	})

	s.T().Run("permission denied for other username", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: jobs[2].Username})
		defer restoreClaims()

		req := &scheduler_proto.ListTasksRequest{
			Username: jobs[0].Username,
		}
		res, err := s.client.ListTasks(ctx, req)
		s.assertGRPCError(err, codes.PermissionDenied)
		s.Nil(res)
	})

	s.T().Run("returns Tasks for requested status", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: jobs[0].Username})
		defer restoreClaims()

		req := &scheduler_proto.ListTasksRequest{
			Username: jobs[0].Username,
			Status:   []scheduler_proto.Task_Status{scheduler_proto.Task_STATUS_NEW},
		}
		res, err := s.client.ListTasks(ctx, req)
		s.Require().NoError(err)
		s.Equal(uint32(1), res.TotalCount)
		s.Equal(tasks[0].ID, res.Tasks[0].Id)
	})

	s.T().Run("returns Tasks for requested statuses job_id order", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: jobs[0].Username})
		defer restoreClaims()

		req := &scheduler_proto.ListTasksRequest{
			Username: jobs[0].Username,
			Status: []scheduler_proto.Task_Status{
				scheduler_proto.Task_STATUS_NEW,
				scheduler_proto.Task_STATUS_FINISHED,
			},
			JobId:    []uint64{jobs[0].ID},
			Ordering: scheduler_proto.ListTasksRequest_CREATED_AT_ASC,
		}
		res, err := s.client.ListTasks(ctx, req)
		s.Require().NoError(err)
		s.Equal(uint32(1), res.TotalCount)
		s.Equal(tasks[0].ID, res.Tasks[0].Id)
		s.Equal(jobs[0].ID, res.Tasks[0].JobId)
	})

	s.T().Run("returns Tasks with limit and offset", func(t *testing.T) {
		restoreClaims := s.claimsInjector.SetClaims(&auth.Claims{Username: jobs[0].Username})
		defer restoreClaims()

		req := &scheduler_proto.ListTasksRequest{
			Username: jobs[0].Username,
			Limit:    1,
			Offset:   1,
		}
		res, err := s.client.ListTasks(ctx, req)
		s.Require().NoError(err)
		s.Equal(uint32(2), res.TotalCount)
		s.Equal(1, len(res.Tasks))
		s.Equal(tasks[0].ID, res.Tasks[0].Id)
	})

}

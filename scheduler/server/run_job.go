package server

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// RunJob creates a new Job. It also initiates immediate scheduling of this Job.
// If immediate scheduling failed for some reason, it waits until this Job is scheduled.
func (s SchedulerServer) RunJob(ctx context.Context, r *scheduler_proto.Job) (*scheduler_proto.Task, error) {
	logger := ctxlogrus.Extract(ctx)
	job := &models.Job{}
	if err := job.FromProto(r); err != nil {
		return nil, errors.Wrap(err, "job.FromProto failed")
	}

	// Check whether the Job request can ever be satisfied.
	if !job.SuitableNodeExists(s.DB) {
		return nil, status.Error(codes.NotFound, "no Nodes satisfy the given constraints")
	}

	publisher, err := events.NewAMQPPublisher(s.AMQPConnection, utils.SchedulerAMQPExchangeName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create NewAMQPPublisher")
	}

	tx1 := s.DB.Begin()
	jobCreatedEvent, err := job.Create(tx1)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new Job")
	}
	if err := utils.SendAndCommit(ctx, tx1, publisher, jobCreatedEvent); err != nil {
		return nil, err
	}

	tx2 := s.DB.Begin()
	node, err := job.FindSuitableNode(tx2)
	if err != nil {
		if txErr := utils.HandleDBError(tx2.Rollback()); txErr != nil {
			return nil, errors.Wrap(txErr, "failed to rollback a transaction for scheduling")
		}
		return s.WaitUntilJobIsScheduled(ctx, job, publisher)
	}

	task, schedulerEvents, err := job.ScheduleForNode(tx2, node)
	if err != nil {
		logger.WithField("job", job).WithField("node", node).
			Infof("unable to create a new Task: %s", err.Error())
		if err := utils.HandleDBError(tx2.Rollback()); err != nil {
			return nil, errors.Wrap(err, "failed to rollback a transaction for scheduling")
		}
		return s.WaitUntilJobIsScheduled(ctx, job, publisher)
	}

	if err := utils.SendAndCommit(ctx, tx2, publisher, schedulerEvents...); err != nil {
		return nil, err
	}
	response, err := task.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "task.ToProto failed")
	}
	return response, nil
}

func (s SchedulerServer) WaitUntilJobIsScheduled(ctx context.Context, job *models.Job, publisher events.Publisher) (*scheduler_proto.Task, error) {
	// Add the Job to the `AwaitingJobs` map.
	s.AwaitingJobs[job.ID] = make(chan *scheduler_proto.Task)

	select {
	case taskProto := <-s.AwaitingJobs[job.ID]:
		// The Job has been scheduled on some Node.
		delete(s.AwaitingJobs, job.ID)
		return taskProto, nil

	case <-ctx.Done():
		// User cancelled the request. Update the Job status.
		delete(s.AwaitingJobs, job.ID)
		tx := s.DB.Begin()
		event, err := job.UpdateStatus(tx, scheduler_proto.Job_STATUS_CANCELLED)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update Job status to CANCELLED")
		}
		subCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := utils.SendAndCommit(subCtx, tx, publisher, event); err != nil {
			return nil, errors.Wrap(err, "failed to send and commit events")
		}
		return nil, status.Error(codes.Canceled, "request was cancelled")
	}
}

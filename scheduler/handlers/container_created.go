package handlers

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// ContainerCreated handles an event when a new Container is created.
type ContainerCreated struct {
	logger           *logrus.Logger
	db               *sqlx.DB
	producer         events.Producer
	commitRetryLimit int
}

// NewContainerCreated creates a new ContainerCreated instance.
func NewContainerCreated(logger *logrus.Logger, db *sqlx.DB, producer events.Producer, commitRetryLimit int) *ContainerCreated {
	return &ContainerCreated{logger: logger, db: db, producer: producer, commitRetryLimit: commitRetryLimit}
}

func (c *ContainerCreated) handle(ctx context.Context, event *events_proto.Event) (err error) {
	eventPayload, ok := event.Payload.(*events_proto.Event_SchedulerContainerCreated)
	if !ok {
		return nil
	}

	c.logger.WithField("event", event.String()).Debugf("processing event %T", eventPayload)
	tx, err := c.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return errors.Wrap(err, "failed to start a transaction")
	}

	container, err := models.NewContainerFromDB(tx, eventPayload.SchedulerContainerCreated.Container.Id)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "failed to get the container from DB"))
	}
	if container.Status != scheduler_proto.Container_NEW {
		c.logger.WithField("container.id", container.Id).Debugf("container has already been scheduled")
		return utils.RollbackTransaction(tx, nil)
	}
	resourceRequest := models.NewResourceRequestFromProto(eventPayload.SchedulerContainerCreated.ResourceRequest)

	nodesExt, err := container.NodesForScheduling(tx, resourceRequest)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.WithStack(err))
	}
	if len(nodesExt) == 0 {
		c.logger.WithField("container.id", container.Id).Debug("no suitable Nodes found to schedule Container")
		return utils.RollbackTransaction(tx, nil)
	}

	models.SortNodesByStrategy(nodesExt, container.SchedulingStrategy, resourceRequest)

	// The Node to schedule the Container to is always the first in the ordered set of Nodes.
	node := nodesExt[0].Node
	nodeUpdates, err := node.AllocateResourcesUpdates(resourceRequest, nil)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.Wrap(err, "failed to allocate resources on the node"))
	}
	if err := node.Update(tx, nodeUpdates); err != nil {
		return utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	containerUpdates := map[string]interface{}{
		"node_id": node.Id,
		"status":  scheduler_proto.Container_SCHEDULED,
	}
	if err := container.Update(tx, containerUpdates); err != nil {
		return utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	containerProto, containerMask, err := container.ToProtoFromUpdates(containerUpdates)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	nodeProto, nodeMask, err := node.ToProtoFromUpdates(nodeUpdates)
	if err != nil {
		return utils.RollbackTransaction(tx, errors.WithStack(err))
	}

	return events.CommitAndPublish(tx, c.producer, &events_proto.Event{
		Payload: &events_proto.Event_SchedulerContainerScheduled{
			SchedulerContainerScheduled: &scheduler_proto.ContainerScheduledEvent{
				Container:       containerProto,
				ContainerMask:   containerMask,
				Node:            nodeProto,
				NodeMask:        nodeMask,
				ResourceRequest: &resourceRequest.ResourceRequest,
			},
		},
		CreatedAt: time.Now().UTC(),
	})
}

// Handle schedules the created Container on an available Node.
func (c *ContainerCreated) Handle(ctx context.Context, event *events_proto.Event) (err error) {
	for i := 0; i <= c.commitRetryLimit; i++ {
		err = errors.Cause(c.handle(ctx, event))
		if err == nil {
			break
		}
		s, ok := status.FromError(err)
		// Retry only if a serializable transaction failed to commit.
		if !ok || s.Code() != codes.Aborted {
			break
		}
	}
	return err
}

package handlers

import (
	"sort"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// ContainerCreated handles an event when a new Container is created.
type ContainerCreated struct {
	logger *logrus.Logger
}

// Handle schedules the created Container on an available Node.
func (c *ContainerCreated) Handle(
	tx utils.SqlxExtGetter,
	eventProto *events_proto.Event,
) ([]*events_proto.Event, error) {
	if eventProto.Type != events_proto.Event_CREATED {
		c.logger.WithField("event", eventProto.String()).
			Debug("Skipping event of a type different than Event_CREATED")
		return nil, nil
	}
	eventContainer, ok := eventProto.Payload.(*events_proto.Event_SchedulerContainer)
	if !ok {
		c.logger.WithField("event", eventProto.String()).Debug("Skipping non Container related event")
		return nil, nil
	}
	c.logger.WithField("event", eventProto.String()).Info("processing a Container created event")
	container := models.NewContainerFromProto(eventContainer.SchedulerContainer)
	resourceRequest, err := models.NewResourceRequestFromDBLatest(tx, container.Id)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	//nodes, err := queries.SelectNodesForContainer(container, resourceRequest)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var nodeSorter sort.Interface

	switch container.SchedulingStrategy {
	case scheduler_proto.Container_CHEAPEST:
		nodeSorter = models.NewNodesByPrice(nodes, resourceRequest)

	case scheduler_proto.Container_LEAST_BUSY:
		nodeSorter = models.NewNodesByAvailability(nodes)

	case scheduler_proto.Container_MOST_RELIABLE:
		nodeSorter = models.NewNodesByReliability(nodes)
	}
	// Sort Nodes by the requested order.
	sort.Sort(nodeSorter)
	node := nodes[0]
	nodeUpdates, err := node.AllocateResources(resourceRequest, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	nodeUpdateEvent, err := node.Update(tx, nodeUpdates)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err := container.Update(tx, map[string]interface{}{
		"node_id": node.Id,
		"status":  scheduler_proto.Container_SCHEDULED,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return []*events_proto.Event{nodeUpdateEvent, containerUpdateEvent}, nil
}

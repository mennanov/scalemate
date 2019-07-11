package handlers_test

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/scheduler/handlers"
	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *HandlersTestSuite) TestContainerCreated_ContainerScheduled() {
	container := &models.Container{}
	s.Require().NoError(container.Create(s.db))
	resourceRequest := &models.ResourceRequest{
		ResourceRequest: scheduler_proto.ResourceRequest{
			ContainerId: container.Id,
		},
	}
	s.Require().NoError(resourceRequest.Create(s.db))
	node := models.Node{
		Node: scheduler_proto.Node{
			Username: "username",
			Name:     "name",
			Status:   scheduler_proto.Node_ONLINE,
		},
	}
	s.Require().NoError(node.Create(s.db))
	nodePricing := models.NodePricing{
		NodePricing: scheduler_proto.NodePricing{
			NodeId: node.Id,
		},
	}
	s.Require().NoError(nodePricing.Create(s.db))

	containerCreatedEvent := &events_proto.Event{
		Payload: &events_proto.Event_SchedulerContainerCreated{
			SchedulerContainerCreated: &scheduler_proto.ContainerCreatedEvent{
				Container:       &container.Container,
				ResourceRequest: &resourceRequest.ResourceRequest,
			},
		},
	}

	handler := handlers.NewContainerCreated(s.logger, s.db, s.producer, 0)
	s.Require().NoError(handler.Handle(context.Background(), containerCreatedEvent))
	s.Require().Len(s.producer.SentEvents, 1)
	event := s.producer.SentEvents[0].Payload.(*events_proto.Event_SchedulerContainerScheduled).SchedulerContainerScheduled
	s.Equal(node.Id, event.Node.Id)
	s.Equal(node.Id, *event.Container.NodeId)
	s.Equal(container.Id, event.Container.Id)
	s.Equal(resourceRequest.Id, event.ResourceRequest.Id)
	// Check idempotency.
	s.Require().NoError(handler.Handle(context.Background(), containerCreatedEvent))
	s.Require().Len(s.producer.SentEvents, 1)
}

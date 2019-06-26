package handlers_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/handlers"
	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *HandlersTestSuite) TestContainerCreated() {
	container := &models.Container{}
	err := container.Create(s.db)
	s.Require().NoError(err)
	resourceRequest := &models.ResourceRequest{
		ResourceRequest: scheduler_proto.ResourceRequest{
			ContainerId: container.Id,
			Cpu:         1,
			Memory:      256,
			Disk:        1024,
			Gpu:         1,
		},
	}
	_, err = resourceRequest.Create(s.db)
	s.Require().NoError(err)

	handler := &handlers.ContainerCreated{}
	newEvents, err := handler.Handle(s.db, containerCreatedEvent)
	s.Require().NoError(err)
	s.Nil(newEvents)
}

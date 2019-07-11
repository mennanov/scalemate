package models_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ModelsTestSuite) TestResourceRequest_Create() {
	container := new(models.Container)
	err := container.Create(s.db)
	s.Require().NoError(err)

	s.Run("empty request", func() {
		s.T().Parallel()
		request := models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
			ContainerId: container.Id,
		})
		s.Require().NoError(request.Create(s.db))
		s.NotNil(request.Id)
		s.False(request.CreatedAt.IsZero())
		s.Nil(request.UpdatedAt)
		// Get the same request from DB and verify they are equal.
		requestFromDB, err := models.NewResourceRequestFromDB(s.db, request.Id)
		s.Require().NoError(err)
		s.Equal(request, requestFromDB)
	})

	s.Run("full request", func() {
		s.T().Parallel()
		request := models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
			ContainerId:   container.Id,
			Cpu:           1,
			Memory:        2,
			Disk:          3,
			Gpu:           4,
			Status:        scheduler_proto.ResourceRequest_CONFIRMED,
			StatusMessage: "confirmed",
		})
		s.Require().NoError(request.Create(s.db))
		s.NotNil(request.Id)
		s.False(request.CreatedAt.IsZero())
		s.Nil(request.UpdatedAt)
		// Get the same request from DB and verify they are equal.
		resourceFromDB, err := models.NewResourceRequestFromDB(s.db, request.Id)
		s.Require().NoError(err)
		s.Equal(request, resourceFromDB)
	})
}

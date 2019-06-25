package models_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ModelsTestSuite) TestResourceRequest_Create() {
	container := new(models.Container)
	_, err := container.Create(s.db)
	s.Require().NoError(err)

	s.Run("empty request", func() {
		s.T().Parallel()
		request := models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
			ContainerId: container.Id,
		})
		event, err := request.Create(s.db)
		s.Require().NoError(err)
		s.NotNil(event)
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
		event, err := request.Create(s.db)
		s.Require().NoError(err)
		s.NotNil(event)
		s.NotNil(request.Id)
		s.False(request.CreatedAt.IsZero())
		s.Nil(request.UpdatedAt)
		// Get the same request from DB and verify they are equal.
		resourceFromDB, err := models.NewResourceRequestFromDB(s.db, request.Id)
		s.Require().NoError(err)
		s.Equal(request, resourceFromDB)
	})
}

func (s *ModelsTestSuite) TestNewResourceRequestFromDBLatest() {
	container := new(models.Container)
	_, err := container.Create(s.db)
	s.Require().NoError(err)

	s.Run("not found", func() {
		request, err := models.NewResourceRequestFromDBLatest(s.db, container.Id)
		testutils.AssertErrorCode(s.T(), err, codes.NotFound)
		s.Nil(request)
	})

	s.Run("returns the most recent ResourceRequest", func() {
		for i := 0; i < 4; i++ {
			recentRequest := models.NewResourceRequestFromProto(&scheduler_proto.ResourceRequest{
				ContainerId: container.Id,
			})
			_, err := recentRequest.Create(s.db)
			s.Require().NoError(err)
			request, err := models.NewResourceRequestFromDBLatest(s.db, container.Id)
			s.Require().NoError(err)
			s.Equal(recentRequest, request)
		}
	})
}

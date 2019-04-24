package models_test

import "github.com/mennanov/scalemate/scheduler/models"

func (s *ModelsTestSuite) TestProcessedEvent_Exists() {
	processedEvent := &models.ProcessedEvent{
		HandlerName: "HandlerName",
		UUID:        []byte{1, 2, 3, 4, 5},
	}
	s.False(processedEvent.Exists(s.gormDB))
	s.Require().NoError(processedEvent.Create(s.gormDB))
	s.True(processedEvent.Exists(s.gormDB))
}

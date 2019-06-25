package models_test

import (
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/scheduler/models"
)

func (s *ModelsTestSuite) TestProcessedEvent_Exists() {
	processedEvent := models.NewProcessedEventFromProto(&events_proto.Event{Uuid: []byte("abcd")})
	s.False(processedEvent.Exists(s.db))
	s.Require().NoError(processedEvent.Create(s.db))
	s.True(processedEvent.Exists(s.db))
}

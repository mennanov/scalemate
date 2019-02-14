package server_test

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

func (s *ServerTestSuite) TestHandleSchedulerNodeCreatedEvent_CreatesNodeInDB() {
	// Create a "node.created" event similar to the one that Scheduler service would create.
	nodeProto := &scheduler_proto.Node{
		Id:          42,
		Username:    "node_username",
		Name:        "node_name",
		CpuModel:    "cpu model",
		MemoryModel: "memory model",
		GpuModel:    "gpu model",
		DiskModel:   "disk model",
	}
	nodeCreatedEvent, err := events.NewEventFromPayload(nodeProto, events_proto.Event_CREATED, events_proto.Service_SCHEDULER, nil)
	s.Require().NoError(err)

	s.Require().NoError(s.service.HandleSchedulerNodeCreatedEvent(nodeCreatedEvent))

	// The Node is expected to be created in DB.
	node := &models.Node{}
	s.Require().NoError(node.Get(s.db, nodeProto.Username, nodeProto.Name))
	s.Equal(nodeProto.Username, node.Username)
	s.Equal(nodeProto.Name, node.Name)
	s.Equal(nodeProto.CpuModel, node.CpuModel)
	s.Equal(nodeProto.MemoryModel, node.MemoryModel)
	s.Equal(nodeProto.GpuModel, node.GpuModel)
	s.Equal(nodeProto.DiskModel, node.DiskModel)
}

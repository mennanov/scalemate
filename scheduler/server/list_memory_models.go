package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// ListMemoryModels lists aggregated GPU models for the given GPU Class.
func (s SchedulerServer) ListMemoryModels(
	ctx context.Context,
	_ *empty.Empty,
) (*scheduler_proto.ListMemoryModelsResponse, error) {

	q := s.DB.Table("nodes").
		Select("memory_model, SUM(memory_capacity) AS memory_capacity, " +
			"SUM(memory_available) AS memory_available, COUNT(id) AS nodes_count").
		Where("status = ?", models.Enum(scheduler_proto.Node_STATUS_ONLINE)).
		Group("memory_model").
		Order("memory_model")

	response := &scheduler_proto.ListMemoryModelsResponse{}

	if err := utils.HandleDBError(q.Scan(&response.MemoryModels)); err != nil {
		return nil, err
	}

	return response, nil

}

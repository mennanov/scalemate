package server

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/shared/utils"
)

// ListGpuModels lists aggregated GPU models for the given GPU Class.
func (s SchedulerServer) ListGpuModels(
	ctx context.Context,
	r *scheduler_proto.ListGpuModelsRequest,
) (*scheduler_proto.ListGpuModelsResponse, error) {
	q := s.db.Table("nodes").
		Select("gpu_model, gpu_class, SUM(gpu_capacity) AS gpu_capacity, SUM(gpu_available) AS gpu_available, " +
			"COUNT(id) AS nodes_count").
		Where("status = ?", utils.Enum(scheduler_proto.Node_STATUS_ONLINE)).
		Group("gpu_class, gpu_model").
		Order("gpu_class, gpu_model")

	if r.GpuClass != 0 {
		q = q.Where("gpu_class = ?", utils.Enum(r.GpuClass))
	}

	response := &scheduler_proto.ListGpuModelsResponse{}

	if err := utils.HandleDBError(q.Scan(&response.GpuModels)); err != nil {
		return nil, err
	}

	return response, nil

}

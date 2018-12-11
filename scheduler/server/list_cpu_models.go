package server

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// ListCpuModels lists aggregated CPU models for the given CPU Class.
func (s SchedulerServer) ListCpuModels(ctx context.Context, r *scheduler_proto.ListCpuModelsRequest) (*scheduler_proto.ListCpuModelsResponse, error) {
	q := s.DB.Table("nodes").
		Select("cpu_model, cpu_class, SUM(cpu_capacity) AS cpu_capacity, SUM(cpu_available) AS cpu_available, " +
			"COUNT(id) AS nodes_count").
		Where("status = ?", models.Enum(scheduler_proto.Node_STATUS_ONLINE)).
		Group("cpu_class, cpu_model").
		Order("cpu_class, cpu_model")

	if r.CpuClass != 0 {
		q = q.Where("cpu_class = ?", models.Enum(r.CpuClass))
	}

	response := &scheduler_proto.ListCpuModelsResponse{}

	if err := utils.HandleDBError(q.Scan(&response.CpuModels)); err != nil {
		return nil, err
	}

	return response, nil

}

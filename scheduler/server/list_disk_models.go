package server

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// ListDiskModels lists aggregated Disk models for the given Disk Class.
func (s SchedulerServer) ListDiskModels(ctx context.Context, r *scheduler_proto.ListDiskModelsRequest) (*scheduler_proto.ListDiskModelsResponse, error) {

	q := s.DB.Table("nodes").
		Select("disk_model, disk_class, SUM(disk_capacity) AS disk_capacity, SUM(disk_available) AS disk_available, " +
			"COUNT(id) AS nodes_count").
		Where("status = ?", models.Enum(scheduler_proto.Node_STATUS_ONLINE)).
		Group("disk_class, disk_model").
		Order("disk_class, disk_model")

	if r.DiskClass != 0 {
		q = q.Where("disk_class = ?", models.Enum(r.DiskClass))
	}

	response := &scheduler_proto.ListDiskModelsResponse{}

	if err := utils.HandleDBError(q.Scan(&response.DiskModels)); err != nil {
		return nil, err
	}

	return response, nil

}

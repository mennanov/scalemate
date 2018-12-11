package server_test

import (
	"context"

	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestCreateJob_MinimalInput() {
	messages, err := events.NewAMQPRawConsumer(s.amqpChannel, events.SchedulerAMQPExchangeName, "", "#")
	s.Require().NoError(err)

	req := &scheduler_proto.Job{
		Username:      "test_username",
		DockerImage:   "postgres:11",
		CpuLimit:      1,
		MemoryLimit:   2000,
		DiskLimit:     1000,
		RestartPolicy: scheduler_proto.Job_RESTART_POLICY_NO,
	}

	res, err := s.client.CreateJob(context.Background(), req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.True(res.Id > 0)

	mask := fieldmask_utils.MaskFromString("Username,DockerImage,CpuLimit,MemoryLimit,DiskLimit,RestartPolicy,Status")
	actual := make(map[string]interface{})
	err = fieldmask_utils.StructToMap(mask, res, actual, stringEye, stringEye)
	s.Require().NoError(err)
	expected := map[string]interface{}{
		"Username":      req.Username,
		"DockerImage":   req.DockerImage,
		"CpuLimit":      req.CpuLimit,
		"MemoryLimit":   req.MemoryLimit,
		"DiskLimit":     req.DiskLimit,
		"RestartPolicy": req.RestartPolicy,
		// Status of a newly created Job must be PENDING.
		"Status": scheduler_proto.Job_STATUS_PENDING,
	}
	s.Equal(expected, actual)
	utils.WaitForMessages(messages, "scheduler.job.created")
}

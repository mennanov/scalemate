package models_test

import (
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes/timestamp"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/genproto/protobuf/field_mask"
)

func (s *ModelsTestSuite) TestTask_FromProto_ToProto() {
	// Create dependent entities first.
	job := &models.Job{
		Username:    "username",
		Status:      models.Enum(scheduler_proto.Job_STATUS_FINISHED),
		DockerImage: "nginx:latest",
	}
	_, err := job.Create(s.db)
	s.Require().NoError(err)
	node := &models.Node{
		Username: "username2",
		Status:   models.Enum(scheduler_proto.Node_STATUS_ONLINE),
	}
	_, err = node.Create(s.db)
	s.Require().NoError(err)

	now := time.Now().Unix()
	testCases := []*scheduler_proto.Task{
		{
			Id:     1,
			JobId:  job.ID,
			NodeId: uint64(node.ID),
			Status: scheduler_proto.Task_STATUS_RUNNING,
			CreatedAt: &timestamp.Timestamp{
				Seconds: now,
			},
			UpdatedAt: &timestamp.Timestamp{
				Seconds: now,
			},
			StartedAt: &timestamp.Timestamp{
				Seconds: now,
			},
			FinishedAt: &timestamp.Timestamp{
				Seconds: now,
			},
			ExitCode:    1,
			ExitMessage: "Exit message",
		},
		{
			Id:     2,
			JobId:  job.ID,
			NodeId: uint64(node.ID),
		},
	}
	mask := fieldmask_utils.MaskFromString(
		"job_id,node_id,status,created_at,updated_at,finished_at,exit_code,exit_message")
	for _, testCase := range testCases {
		task := &models.Task{}
		err := task.FromProto(testCase)
		s.Require().NoError(err)
		// Create the task in DB.
		_, err = task.Create(s.db)
		s.Require().NoError(err)
		// Retrieve the same task from DB.
		taskFromDB := &models.Task{Model: models.Model{ID: task.ID}}
		s.Require().NoError(taskFromDB.LoadFromDB(s.db))
		taskFromDBProto, err := taskFromDB.ToProto(nil)
		s.Require().NoError(err)

		expected := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, taskFromDBProto, expected, generator.CamelCase, stringEye)
		s.Require().NoError(err)

		actual := make(map[string]interface{})
		err = fieldmask_utils.StructToMap(mask, taskFromDBProto, actual, generator.CamelCase, stringEye)
		s.Require().NoError(err)

		s.Equal(expected, actual)
	}
}

func (s *ModelsTestSuite) TestTask_ToProto() {
	task := &models.Task{
		Model:       models.Model{ID: 42},
		JobID:       1,
		NodeID:      2,
		Status:      models.Enum(scheduler_proto.Task_STATUS_RUNNING),
		ExitCode:    1,
		ExitMessage: "Exit message",
	}
	mask := &field_mask.FieldMask{Paths: []string{"id", "job_id", "node_id"}}
	taskProto, err := task.ToProto(mask)
	s.Require().NoError(err)
	s.Equal(uint64(42), taskProto.Id)
	s.Equal(uint64(1), taskProto.JobId)
	s.Equal(uint64(2), taskProto.NodeId)
	// Values below are expected to be default for those types as they are not in the mask.
	s.Equal(int32(0), taskProto.ExitCode)
	s.Equal("", taskProto.ExitMessage)
}

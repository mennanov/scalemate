package models_test

import (
	"time"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
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
		},
		{
			Id:     2,
			JobId:  job.ID,
			NodeId: uint64(node.ID),
		},
	}
	mask := fieldmask_utils.MaskFromString(
		"job_id,node_id,status,created_at,updated_at,finished_at")
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
		Model:  models.Model{ID: 42},
		JobID:  1,
		NodeID: 2,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	mask := &field_mask.FieldMask{Paths: []string{"id", "job_id", "node_id"}}
	taskProto, err := task.ToProto(mask)
	s.Require().NoError(err)
	s.Equal(uint64(42), taskProto.Id)
	s.Equal(uint64(1), taskProto.JobId)
	s.Equal(uint64(2), taskProto.NodeId)
}

func (s *ModelsTestSuite) TestTasks_UpdateForDisconnectedNode_UpdatesOnlyRunningTasksForThatNode() {
	s.db = s.db.LogMode(true)

	node1 := &models.Node{
		Username: "username",
		Name:     "node1",
	}
	_, err := node1.Create(s.db)
	s.Require().NoError(err)

	node2 := &models.Node{
		Username: "username",
		Name:     "node2",
	}
	_, err = node2.Create(s.db)
	s.Require().NoError(err)

	job1 := &models.Job{}
	_, err = job1.Create(s.db)
	s.Require().NoError(err)

	job2 := &models.Job{}
	_, err = job2.Create(s.db)
	s.Require().NoError(err)

	taskRunningOnNode1 := &models.Task{
		JobID:  job1.ID,
		NodeID: node1.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = taskRunningOnNode1.Create(s.db)
	s.Require().NoError(err)

	taskFinishedOnNode1 := &models.Task{
		JobID:  job1.ID,
		NodeID: node1.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_FINISHED),
	}
	_, err = taskFinishedOnNode1.Create(s.db)
	s.Require().NoError(err)

	taskRunningOnNode2 := &models.Task{
		JobID:  job2.ID,
		NodeID: node2.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = taskRunningOnNode2.Create(s.db)
	s.Require().NoError(err)

	var tasks models.Tasks
	updateEvents, err := tasks.UpdateStatusForDisconnectedNode(s.db, node1.ID)
	s.Require().NoError(err)
	s.Require().Len(updateEvents, 1)
	s.Require().Len(tasks, 1)
	taskProtoMsg, err := utils.NewModelProtoFromEvent(updateEvents[0])
	s.Require().NoError(err)
	taskProto, ok := taskProtoMsg.(*scheduler_proto.Task)
	s.Require().True(ok)
	s.Equal(taskRunningOnNode1.ID, taskProto.Id)
	s.Equal(scheduler_proto.Task_STATUS_NODE_FAILED, taskProto.Status)
	// Status should be updated only for taskRunningOnNode1.
	s.Require().NoError(taskRunningOnNode1.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Task_STATUS_NODE_FAILED), taskRunningOnNode1.Status)
	// tasks should contain exactly 1 element by that time.
	s.Equal(taskRunningOnNode1.ID, tasks[0].ID)
	s.Equal(taskRunningOnNode1.Status, tasks[0].Status)
	s.Require().NoError(taskFinishedOnNode1.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Task_STATUS_FINISHED), taskFinishedOnNode1.Status)
	s.Require().NoError(taskRunningOnNode2.LoadFromDB(s.db))
	s.Equal(models.Enum(scheduler_proto.Task_STATUS_RUNNING), taskRunningOnNode2.Status)
}

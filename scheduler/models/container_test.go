package models_test

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ModelsTestSuite) TestContainer_FromProto_Create_ToProto() {
	now := time.Now().Unix()
	testCases := []struct {
		containerProto *scheduler_proto.Container
		mask           fieldmask_utils.FieldFilter
	}{
		{
			containerProto: &scheduler_proto.Container{
				Username:      "username",
				Status:        scheduler_proto.Container_CANCELLED,
				StatusMessage: "status message",
				Image:         "image",
				CpuClassMin:   scheduler_proto.CPUClass_CPU_CLASS_ENTRY,
				CpuClassMax:   scheduler_proto.CPUClass_CPU_CLASS_ADVANCED,
				GpuClassMin:   scheduler_proto.GPUClass_GPU_CLASS_ENTRY,
				GpuClassMax:   scheduler_proto.GPUClass_GPU_CLASS_PRO,
				DiskClassMin:  scheduler_proto.DiskClass_DISK_CLASS_HDD,
				DiskClassMax:  scheduler_proto.DiskClass_DISK_CLASS_SSD,
				CreatedAt: &timestamp.Timestamp{
					Seconds: now,
				},
				UpdatedAt: &timestamp.Timestamp{
					Seconds: now,
				},
				Labels:         []string{"label1", "label2"},
				AgentAuthToken: []byte("token"),
			},
			mask: fieldmask_utils.MaskInverse{"Id": nil},
		},
		//{
		//	containerProto: &scheduler_proto.Container{
		//		Username: "username",
		//	},
		//	mask: fieldmask_utils.MaskFromString("Username"),
		//},
	}

	for _, testCase := range testCases {
		container, err := models.NewContainerFromProto(testCase.containerProto)
		s.Require().NoError(err)
		containerProto, err := container.ToProto(nil)
		s.Require().NoError(err)
		s.Require().True(reflect.DeepEqual(testCase.containerProto, containerProto))

		// Create the container in DB to verify that everything is persisted.
		_, err = container.Create(s.gormDB)
		s.Require().NoError(err)

		// Retrieve the same container from DB.
		containerFromDB := &models.Container{}
		s.gormDB.First(containerFromDB, container.ID)
		containerFromDBProto, err := containerFromDB.ToProto(nil)
		s.Require().NoError(err)

		// Proto message for the Container from DB differs from the original one in the test case as there are some
		// values added to the Container when it's saved.
		protoFiltered := &scheduler_proto.Container{}
		fmt.Println("containerFromDBProto:", containerFromDBProto)
		fmt.Println("testCase.containerProto:", testCase.containerProto)
		err = fieldmask_utils.StructToStruct(testCase.mask, containerFromDBProto, protoFiltered)
		s.Require().NoError(err)
		s.Equal(testCase.containerProto, protoFiltered)
		s.True(reflect.DeepEqual(testCase.containerProto, protoFiltered))
	}
}

func (s *ModelsTestSuite) TestContainer_UpdateStatus() {
	for statusFrom, statusesTo := range models.ContainerStatusTransitions {
		for _, statusTo := range statusesTo {
			container := &models.Container{Status: utils.Enum(statusFrom)}
			_, err := container.Create(s.gormDB)
			s.Require().NoError(err)
			s.Nil(container.UpdatedAt)

			_, err = container.UpdateStatus(s.gormDB, statusTo)
			s.Require().NoError(err)
			s.NotNil(container.UpdatedAt)
			s.Equal(container.Status, utils.Enum(statusTo))
			// Verify that the status is actually persisted in DB.
			s.Require().NoError(container.LoadFromDB(s.gormDB))
			s.Equal(container.Status, utils.Enum(statusTo))
		}
	}
}

func (s *ModelsTestSuite) TestContainer_StatusTransitions() {
	for status, name := range scheduler_proto.Container_Status_name {
		_, ok := models.ContainerStatusTransitions[scheduler_proto.Container_Status(status)]
		s.True(ok, "%s not found in models.ContainerStatusTransitions", name)
	}
}

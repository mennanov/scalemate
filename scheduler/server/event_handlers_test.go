package server_test

import (
	"context"
	"sync"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

func (s *ServerTestSuite) TestHandleNodeConnectedEvents() {
	// Create a "scheduler.node.updated" event similar to the one that Scheduler service would create.
	node := &models.Node{
		Username:        "username",
		Name:            "node_name",
		Status:          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		CpuCapacity:     4,
		CpuAvailable:    2,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		MemoryCapacity:  32000,
		MemoryAvailable: 16000,
		DiskCapacity:    64000,
		DiskAvailable:   28000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	jobs := []*models.Job{
		// Both Jobs fit into the Node above.
		{
			Username:    "job1",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    1,
			CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
			MemoryLimit: 8000,
			DiskLimit:   14000,
			DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		},
		{
			Username:    "job2",
			Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
			CpuLimit:    1,
			CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
			MemoryLimit: 8000,
			DiskLimit:   14000,
			DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		},
	}

	publisher, err := events.NewAMQPPublisher(s.service.AMQPConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	sg := &sync.WaitGroup{}
	sg.Add(len(jobs))

	var jobIds []uint64
	for _, job := range jobs {
		_, err := job.Create(s.service.DB)
		s.Require().NoError(err)

		jobIds = append(jobIds, job.ID)

		// Wait for that job to be scheduled
		go func(job *models.Job, w *sync.WaitGroup) {
			taskProto, err := s.service.WaitUntilJobIsScheduled(context.Background(), job, publisher)
			s.Require().NoError(err)
			s.Contains(jobIds, taskProto.JobId)
			w.Done()
		}(job, sg)
	}

	err = s.service.ConnectNode(context.Background(), node, publisher)
	s.Require().NoError(err)

	// Node receives tasks.
	sg.Add(1)
	go func(w *sync.WaitGroup) {
		// Receive all the Tasks for the Node.
		for range jobs {
			taskProto := <-s.service.ConnectedNodes[node.ID]
			s.Contains(jobIds, taskProto.JobId)
			s.Equal(node.ID, taskProto.NodeId)
		}
		w.Done()
	}(sg)

	// HandleNodeConnectedEvents is expected to create new Tasks for the existing Jobs.
	// Afterwards HandleTaskCreatedEvents is expected to send those Tasks to the appropriate channels.
	sg.Wait()
	// There should be no awaiting Jobs at that moment.
	s.Equal(0, len(s.service.AwaitingJobs))
	// Reload Node from DB.
	s.Require().NoError(node.LoadFromDB(s.service.DB))
	// Jobs are expected to acquire all Node's resources.
	s.Equal(float32(0), node.CpuAvailable)
	s.Equal(uint32(0), node.DiskAvailable)
	s.Equal(uint32(0), node.GpuAvailable)
	s.Equal(uint32(0), node.MemoryAvailable)
	// Verify that Jobs now have a status "SCHEDULED".
	for _, job := range jobs {
		s.Require().NoError(job.LoadFromDB(s.service.DB))
		s.Equal(models.Enum(scheduler_proto.Job_STATUS_SCHEDULED), job.Status)
	}
}

func (s *ServerTestSuite) TestHandleTaskStatusUpdatedEvents() {
	// Create a "scheduler.node.updated" event similar to the one that Scheduler service would create.
	node := &models.Node{
		Username:        "username",
		Name:            "node_name",
		Status:          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		CpuCapacity:     4,
		CpuAvailable:    2,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		GpuCapacity:     2,
		GpuAvailable:    2,
		GpuClass:        models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
		GpuClassMin:     models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
		MemoryCapacity:  32000,
		MemoryAvailable: 16000,
		DiskCapacity:    64000,
		DiskAvailable:   28000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)
	fakePublisher := events.NewFakePublisher()
	// Connect the Node without sending an actual event about it.
	s.Require().NoError(s.service.ConnectNode(context.Background(), node, fakePublisher))

	jobPending := &models.Job{
		Username:    "job1",
		Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    1,
		CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		GpuLimit:    1,
		GpuClass:    models.Enum(scheduler_proto.GPUClass_GPU_CLASS_INTERMEDIATE),
		MemoryLimit: 8000,
		DiskLimit:   14000,
		DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
	}
	_, err = jobPending.Create(s.service.DB)
	s.Require().NoError(err)

	sg := &sync.WaitGroup{}
	sg.Add(1)

	go func(w *sync.WaitGroup) {
		// Receive the Task for the Node.
		taskProto := <-s.service.ConnectedNodes[node.ID]
		s.Equal(jobPending.ID, taskProto.JobId)
		s.Equal(node.ID, taskProto.NodeId)
		w.Done()
	}(sg)

	publisher, err := events.NewAMQPPublisher(s.service.AMQPConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	sg.Add(1)
	go func(w *sync.WaitGroup) {
		// Receive the Task for the Job.
		taskProto, err := s.service.WaitUntilJobIsScheduled(context.Background(), jobPending, publisher)
		s.Require().NoError(err)
		s.Equal(jobPending.ID, taskProto.JobId)
		s.Equal(node.ID, taskProto.NodeId)
		s.Equal(scheduler_proto.Task_STATUS_UNKNOWN, taskProto.Status)
		w.Done()
	}(sg)

	jobScheduled := &models.Job{
		Username:    "job2",
		Status:      models.Enum(scheduler_proto.Job_STATUS_SCHEDULED),
		CpuLimit:    2,
		CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		MemoryLimit: 16000,
		DiskLimit:   36000,
		DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
	}
	_, err = jobScheduled.Create(s.service.DB)
	s.Require().NoError(err)

	taskRunning := &models.Task{
		JobID:  jobScheduled.ID,
		NodeID: node.ID,
		Status: models.Enum(scheduler_proto.Task_STATUS_RUNNING),
	}
	_, err = taskRunning.Create(s.service.DB)
	s.Require().NoError(err)

	// Simulate the Task finished event.
	event, err := taskRunning.UpdateStatus(s.service.DB, scheduler_proto.Task_STATUS_FINISHED)
	s.Require().NoError(err)
	s.Require().NoError(publisher.Send(event))

	// HandleTaskCreatedEvents is expected to schedule pending Jobs onto the Node where the Task has finished running.
	sg.Wait()
	// There should be no awaiting Jobs at that moment.
	s.Equal(0, len(s.service.AwaitingJobs))
	// Reload Node from DB.
	s.Require().NoError(node.LoadFromDB(s.service.DB))
	s.Equal(float32(1), node.CpuAvailable)
	s.Equal(uint32(14000), node.DiskAvailable)
	s.Equal(uint32(1), node.GpuAvailable)
	s.Equal(uint32(8000), node.MemoryAvailable)
	// Verify that the jobPending now has a status "SCHEDULED".
	s.Require().NoError(jobPending.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_SCHEDULED), jobPending.Status)
}

func (s *ServerTestSuite) TestHandleTaskCreatedEvents() {
	node := &models.Node{
		Username:        "username",
		Name:            "node_name",
		Status:          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
		CpuCapacity:     4,
		CpuAvailable:    2,
		CpuClass:        models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		CpuClassMin:     models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		MemoryCapacity:  32000,
		MemoryAvailable: 16000,
		DiskCapacity:    64000,
		DiskAvailable:   28000,
		DiskClass:       models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
		DiskClassMin:    models.Enum(scheduler_proto.DiskClass_DISK_CLASS_HDD),
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)
	fakePublisher := events.NewFakePublisher()
	// Connect the Node without sending an actual event about it.
	s.Require().NoError(s.service.ConnectNode(context.Background(), node, fakePublisher))

	job := &models.Job{
		Username:    "job",
		Status:      models.Enum(scheduler_proto.Job_STATUS_PENDING),
		CpuLimit:    2,
		CpuClass:    models.Enum(scheduler_proto.CPUClass_CPU_CLASS_ADVANCED),
		MemoryLimit: 16000,
		DiskLimit:   28000,
		DiskClass:   models.Enum(scheduler_proto.DiskClass_DISK_CLASS_SSD),
	}
	_, err = job.Create(s.service.DB)
	s.Require().NoError(err)

	publisher, err := events.NewAMQPPublisher(s.service.AMQPConnection, utils.SchedulerAMQPExchangeName)
	s.Require().NoError(err)

	sg := &sync.WaitGroup{}
	sg.Add(1)

	go func(w *sync.WaitGroup) {
		// Receive the Task for the Node.
		taskProto := <-s.service.ConnectedNodes[node.ID]
		s.Equal(job.ID, taskProto.JobId)
		s.Equal(node.ID, taskProto.NodeId)
		w.Done()
	}(sg)

	sg.Add(1)
	go func(w *sync.WaitGroup) {
		// Receive the Task for the Job.
		taskProto, err := s.service.WaitUntilJobIsScheduled(context.Background(), job, publisher)
		s.Require().NoError(err)
		s.Equal(job.ID, taskProto.JobId)
		s.Equal(node.ID, taskProto.NodeId)
		s.Equal(scheduler_proto.Task_STATUS_UNKNOWN, taskProto.Status)
		w.Done()
	}(sg)

	_, schedulingEvents, err := job.ScheduleForNode(s.service.DB, node)
	s.Require().NoError(err)
	s.Require().NoError(publisher.Send(schedulingEvents...))

	sg.Wait()
	s.Equal(0, len(s.service.AwaitingJobs))
	// Reload Node from DB.
	s.Require().NoError(node.LoadFromDB(s.service.DB))
	s.Equal(float32(0), node.CpuAvailable)
	s.Equal(uint32(0), node.DiskAvailable)
	s.Equal(uint32(0), node.GpuAvailable)
	s.Equal(uint32(0), node.MemoryAvailable)
	// Verify that the job now has a status "SCHEDULED".
	s.Require().NoError(job.LoadFromDB(s.service.DB))
	s.Equal(models.Enum(scheduler_proto.Job_STATUS_SCHEDULED), job.Status)
}

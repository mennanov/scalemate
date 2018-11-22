package server

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jinzhu/gorm"
	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/events/events_proto"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	// ErrNodeAlreadyConnected is used when the Node can't be "connected" because it is already connected.
	ErrNodeAlreadyConnected = errors.New("Node is already connected")
	// ErrNodeIsNotConnected is used when the Node can't be disconnected because it's already disconnected.
	ErrNodeIsNotConnected = errors.New("Node is not connected")
)

const (
	// NodeConnectedEventsQueueName is the name of the AMQP queue to be used to receive events about the newly connected
	// Nodes.
	// This is intended to be a "work" queue: a single queue with multiple consumers to make sure only 1 consumer
	// receives the event at a time (simple round robin).
	NodeConnectedEventsQueueName = "scheduler_node_connected"

	// TaskStatusUpdatedEventsQueueName is the name of the AMQP queue to be used to receive events about Tasks status
	// change. We are interested only in those messages when the Task's status is changed to FINISHED, so all the
	// messages in the queue have to be processed and filtered.
	// This is intended to be a "work" queue: a single queue with multiple consumers to make sure only 1 consumer
	// receives the event at a time (simple round robin).
	TaskStatusUpdatedEventsQueueName = "scheduler_task_status_updated"
)

// ConnectNode adds the Node ID to the map of connected Nodes and updates the Node's status in DB.
// This method should be called when the Node connects to receive Tasks.
func (s *SchedulerServer) ConnectNode(ctx context.Context, node *models.Node, publisher events.Publisher) error {
	if _, ok := s.ConnectedNodes[node.ID]; ok {
		return ErrNodeAlreadyConnected
	}
	tx := s.DB.Begin()
	event, err := node.Updates(tx, map[string]interface{}{
		"connected_at": time.Now(),
		"status":       models.Enum(scheduler_proto.Node_STATUS_ONLINE),
	})
	if err != nil {
		return err
	}
	if err := utils.SendAndCommit(ctx, tx, publisher, event); err != nil {
		return err
	}
	s.ConnectedNodes[node.ID] = make(chan *scheduler_proto.Task)
	return nil
}

// DisconnectNode removes the Node ID from the map of connected Nodes and updates the Node's status in DB.
// This method should be called when the Node disconnects.
func (s *SchedulerServer) DisconnectNode(ctx context.Context, node *models.Node, publisher events.Publisher) error {
	if _, ok := s.ConnectedNodes[node.ID]; !ok {
		return ErrNodeIsNotConnected
	}
	tx := s.DB.Begin()
	event, err := node.Updates(tx, map[string]interface{}{
		"disconnected_at": time.Now(),
		"status":          models.Enum(scheduler_proto.Node_STATUS_OFFLINE),
	})
	if err != nil {
		return err
	}
	if err := utils.SendAndCommit(ctx, tx, publisher, event); err != nil {
		return err
	}

	delete(s.ConnectedNodes, node.ID)
	return nil
}

// HandleNodeConnectedEvents handles the event when a Node connects to receive Tasks.
// A subset of Jobs that have not been scheduled yet are selected to be scheduled on this Node.
func (s *SchedulerServer) HandleNodeConnectedEvents() error {
	amqpChannel, err := s.AMQPConnection.Channel()
	defer utils.Close(amqpChannel)
	if err != nil {
		return errors.Wrap(err, "failed to create AMQP channel")
	}

	// Declare a "work" queue to be used by multiple consumers (other Scheduler service instances).
	queue, err := amqpChannel.QueueDeclare(
		NodeConnectedEventsQueueName,
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return errors.Wrapf(err, "failed to declare AMQP queue %s", NodeConnectedEventsQueueName)
	}

	// Get all Node UPDATED events where the field "connected_at" was updated which indicates that the Node is connected.
	key := "scheduler.node.updated.#.connected_at.#"
	if err = amqpChannel.QueueBind(
		queue.Name,
		key,
		utils.SchedulerAMQPExchangeName,
		false,
		nil); err != nil {
		return errors.Wrapf(err, "failed to bind AMQP Queue for key '%s'", key)
	}

	messages, err := amqpChannel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil)

	if err != nil {
		return errors.Wrap(err, "failed to register AMQP Consumer")
	}

	publisher, err := events.NewAMQPPublisher(s.AMQPConnection, utils.SchedulerAMQPExchangeName)
	if err != nil {
		return errors.Wrap(err, "failed to create NewAMQPPublisher")
	}

	for msg := range messages {
		// Process the message in a non-blocking way.
		go func(msg amqp.Delivery) {
			defer msg.Ack(false)
			eventProto := &events_proto.Event{}
			if err := proto.Unmarshal(msg.Body, eventProto); err != nil {
				logrus.Error("failed to unmarshal events_proto.Event")
				return
			}
			m, err := events.NewModelProtoFromEvent(eventProto)
			if err != nil {
				logrus.Errorf("NewModelProtoFromEvent in HandleNodeConnectedEvents failed: %s", err.Error())
				return
			}
			nodeProto, ok := m.(*scheduler_proto.Node)
			if !ok {
				logrus.WithField("modelProto", m.String()).
					Errorf("failed to convert message event proto to model proto")
				return
			}
			node := &models.Node{}
			if err := node.FromProto(nodeProto); err != nil {
				logrus.WithField("modelProto", m.String()).
					Errorf("failed to convert message event proto to model proto")
				return
			}
			// Populate the node struct fields from DB.
			if err := node.LoadFromDB(s.DB); err != nil {
				logrus.WithError(err).Error("failed to get a Node by ID")
				return
			}
			if err := scheduleJobsForNode(node, s.DB, publisher); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"nodeId": node.ID,
				}).Info("failed to schedule jobs for recently connected Node")
				return
			}
		}(msg)
	}
	return nil
}

// HandleTaskStatusUpdatedEvents handles the event when the Task execution is finished successfully and the Node that
// this Task was running on has some resources available for new Jobs.
func (s *SchedulerServer) HandleTaskStatusUpdatedEvents() error {
	amqpChannel, err := s.AMQPConnection.Channel()
	defer utils.Close(amqpChannel)
	if err != nil {
		return errors.Wrap(err, "failed to create AMQP channel")
	}

	// Declare a "work" queue to be used by multiple consumers (other Scheduler service instances).
	queue, err := amqpChannel.QueueDeclare(
		TaskStatusUpdatedEventsQueueName,
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return errors.Wrapf(err, "failed to declare AMQP queue %s", TaskStatusUpdatedEventsQueueName)
	}

	// Get all Task UPDATED events where the field "status" was updated.
	key := "scheduler.task.updated.#.status.#"
	if err = amqpChannel.QueueBind(
		queue.Name,
		key,
		utils.SchedulerAMQPExchangeName,
		false,
		nil); err != nil {
		return errors.Wrapf(err, "failed to bind AMQP Queue for key '%s'", key)
	}

	messages, err := amqpChannel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil)

	if err != nil {
		return errors.Wrap(err, "failed to register AMQP Consumer")
	}

	for msg := range messages {
		// Process the message in a non-blocking way.
		go func(msg amqp.Delivery) {
			defer msg.Ack(false)
			eventProto := &events_proto.Event{}
			if err := proto.Unmarshal(msg.Body, eventProto); err != nil {
				logrus.Error("failed to unmarshal events_proto.Event")
				return
			}
			m, err := events.NewModelProtoFromEvent(eventProto)
			if err != nil {
				logrus.WithError(err).Error("NewModelProtoFromEvent in HandleTaskStatusUpdatedEvents failed")
				return
			}
			taskProto, ok := m.(*scheduler_proto.Task)
			if !ok {
				logrus.WithField("modelProto", m.String()).
					Error("failed to convert message event proto to *scheduler_proto.Task")
				return
			}
			// Verify that the Task has terminated.
			if taskProto.Status != scheduler_proto.Task_STATUS_FINISHED &&
				taskProto.Status != scheduler_proto.Task_STATUS_FAILED &&
				taskProto.Status != scheduler_proto.Task_STATUS_NODE_FAILED {
				// Acknowledge the message and terminate.
				msg.Ack(false)
				return
			}
			task := &models.Task{}
			if err := task.FromProto(taskProto); err != nil {
				logrus.WithField("taskProto", taskProto.String()).WithError(err).
					Error("task.FromProto failed")
				return
			}
			// Populate the task struct fields from DB.
			if err := task.LoadFromDB(s.DB, "node_id"); err != nil {
				logrus.WithError(err).Error("failed to get a Task by ID")
				return
			}
			node := &models.Node{
				Model: models.Model{ID: task.NodeID},
			}
			// Populate the node struct fields from DB.
			if err := node.LoadFromDB(s.DB); err != nil {
				logrus.WithError(err).Error("failed to get a Node by ID")
				return
			}
			publisher, err := events.NewAMQPPublisher(s.AMQPConnection, utils.SchedulerAMQPExchangeName)
			if err != nil {
				logrus.Errorf("failed to create NewAMQPPublisher: %s", err.Error())
				return
			}
			if err := scheduleJobsForNode(node, s.DB, publisher); err != nil {
				logrus.WithError(err).WithFields(logrus.Fields{
					"nodeId": node.ID,
				}).Info("failed to schedule jobs for Node")
				return
			}
		}(msg)
	}
	return nil
}

// HandleTaskCreatedEvents handles the event when the Task is created. The handler checks whether the Node to which the
// Task has been scheduled to is connected to this Scheduler service instance. If not: disregards the event, otherwise
// sends the Task to the channel where it will be picked by the Node to run it.
func (s *SchedulerServer) HandleTaskCreatedEvents() error {
	amqpChannel, err := s.AMQPConnection.Channel()
	defer utils.Close(amqpChannel)
	if err != nil {
		return errors.Wrap(err, "failed to create AMQP channel")
	}

	// Temp queue because it is only needed when this particular service instance is up and connected to AMQP.
	queue, err := amqpChannel.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil)
	if err != nil {
		return errors.Wrapf(err, "failed to declare AMQP queue %s", TaskStatusUpdatedEventsQueueName)
	}

	// Get all Task CREATED events.
	key := "scheduler.task.created"
	if err = amqpChannel.QueueBind(
		queue.Name,
		key,
		utils.SchedulerAMQPExchangeName,
		false,
		nil); err != nil {
		return errors.Wrapf(err, "failed to bind AMQP Queue for key '%s'", key)
	}

	messages, err := amqpChannel.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil)

	if err != nil {
		return errors.Wrap(err, "failed to register AMQP Consumer")
	}

	for msg := range messages {
		// Process the message in a non-blocking way.
		go func(msg amqp.Delivery) {
			// Always acknowledge the message because it can only fail in a non-retriable way.
			defer msg.Ack(false)
			eventProto := &events_proto.Event{}
			if err := proto.Unmarshal(msg.Body, eventProto); err != nil {
				logrus.WithError(err).Error("failed to unmarshal events_proto.Event")
				return
			}
			m, err := events.NewModelProtoFromEvent(eventProto)
			if err != nil {
				logrus.WithError(err).Error("NewModelProtoFromEvent in HandleNodeConnectedEvents failed")
				return
			}
			taskProto, ok := m.(*scheduler_proto.Task)
			if !ok {
				logrus.WithField("modelProto", m.String()).
					Error("failed to convert message event proto to *scheduler_proto.Task")
				return
			}
			if ch, ok := s.ConnectedNodes[taskProto.NodeId]; ok {
				// Send the Task to the Node that it is scheduled for.
				ch <- taskProto
			}
			if ch, ok := s.AwaitingJobs[taskProto.JobId]; ok {
				// Send the Task to the waiting client.
				ch <- taskProto
			}
		}(msg)
	}
	return nil
}

// scheduleJobsForNode finds suitable Jobs that can be scheduled on the given Node and selects the best combination of
// these Jobs so that they all fit into the Node.
func scheduleJobsForNode(node *models.Node, db *gorm.DB, publisher events.Publisher) error {
	tx := db.Begin()
	jobs, err := node.FindSuitableJobs(tx)
	if err != nil {
		return errors.Wrap(err, "no suitable Jobs could be found for the recently connected Node")
	}

	res := models.AvailableResources{
		CpuAvailable:    node.CpuAvailable,
		MemoryAvailable: node.MemoryAvailable,
		GpuAvailable:    node.GpuAvailable,
		DiskAvailable:   node.DiskAvailable,
	}
	// selectedJobs is a bitarray.BitArray of the selected for scheduling Jobs.
	selectedJobs := models.SelectJobs(jobs, res)
	allEvents := make([]*events_proto.Event, 0)
	for i, job := range jobs {
		if !selectedJobs.GetBit(uint64(i)) {
			// This Job has not been selected for scheduling: disregard it.
			continue
		}
		_, schedulerEvents, err := job.ScheduleForNode(tx, node)
		if err != nil {
			logrus.WithField("job", job).WithField("node", node).
				Infof("unable to schedule Job for Node: %s", err.Error())
			if err := utils.HandleDBError(tx.Rollback()); err != nil {
				return errors.Wrap(err, "failed to rollback transaction")
			}
		}
		allEvents = append(allEvents, schedulerEvents...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// Send all scheduling events and commit the transaction.
	if err := utils.SendAndCommit(ctx, tx, publisher, allEvents...); err != nil {
		return errors.Wrap(err, "failed to send events and commit transaction")
	}
	return nil
}

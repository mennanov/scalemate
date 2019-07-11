package backend

import (
	"context"
	"database/sql"
	"time"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/scheduler/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// CreateNode creates a new Node in DB.
func (s *SchedulerBackend) CreateNode(ctx context.Context, request *scheduler_proto.NodeWithPricing) (*scheduler_proto.NodeWithPricing, error) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get auth claims")
	}

	if err := ValidateNodeFields(request.Node); err != nil {
		return nil, errors.WithStack(err)
	}
	if err := ValidateNodePricingFields(request.NodePricing); err != nil {
		return nil, errors.WithStack(err)
	}

	request.Node.Username = claims.Username
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, errors.Wrap(err, "failed to start a transaction")
	}

	node := models.NewNodeFromProto(request.Node)
	if err := node.Create(tx); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to create a new node"))
	}

	nodePricing := models.NewNodePricingFromProto(request.NodePricing)
	nodePricing.NodeId = node.Id
	if err := nodePricing.Create(tx); err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to create a new node pricing"))
	}

	if err := events.CommitAndPublish(tx, s.producer, &events_proto.Event{
		Payload: &events_proto.Event_SchedulerNodeCreated{
			SchedulerNodeCreated: &scheduler_proto.NodeCreatedEvent{
				Node:        &node.Node,
				NodePricing: &nodePricing.NodePricing,
			},
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, err
	}

	return &scheduler_proto.NodeWithPricing{
		Node:        &node.Node,
		NodePricing: &nodePricing.NodePricing,
	}, nil
}

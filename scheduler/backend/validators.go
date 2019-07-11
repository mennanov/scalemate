package backend

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/utils"
)

// ValidateNodeFields validates fields for a new Node.
func ValidateNodeFields(node *scheduler_proto.Node) error {
	if node == nil {
		return status.Error(codes.InvalidArgument, "Node is nil")
	}
	rules := []*validation.FieldRules{
		validation.Field(&node.Id, utils.ValidateIsEmpty),
		validation.Field(&node.Status, utils.ValidateIsEmpty),
		validation.Field(&node.Username, utils.ValidateIsEmpty),
		validation.Field(&node.ConnectedAt, utils.ValidateIsEmpty),
		validation.Field(&node.DisconnectedAt, utils.ValidateIsEmpty),
		validation.Field(&node.CreatedAt, utils.ValidateIsEmpty),
		validation.Field(&node.UpdatedAt, utils.ValidateIsEmpty),
		validation.Field(&node.Fingerprint, validation.Required),
	}

	if err := validation.ValidateStruct(node, rules...); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

// ValidateNodePricingFields validates fields for a new NodePricing.
func ValidateNodePricingFields(nodePricing *scheduler_proto.NodePricing) error {
	if nodePricing == nil {
		return status.Error(codes.InvalidArgument, "NodePricing is nil")
	}
	rules := []*validation.FieldRules{
		validation.Field(&nodePricing.Id, utils.ValidateIsEmpty),
		validation.Field(&nodePricing.CreatedAt, utils.ValidateIsEmpty),
	}

	if err := validation.ValidateStruct(nodePricing, rules...); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

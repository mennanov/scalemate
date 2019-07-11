package frontend

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/gogo/protobuf/types"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/utils"
)

// ValidateContainerFields checks if the Container fields are valid and can be used to create a new Container.
func ValidateContainerFields(c *scheduler_proto.Container) error {
	if c == nil {
		return status.Error(codes.InvalidArgument, "Container is nil")
	}
	if err := validation.ValidateStruct(c,
		validation.Field(&c.Id, utils.ValidateIsEmpty),
		validation.Field(&c.Username, utils.ValidateIsEmpty),
		validation.Field(&c.NodeId, utils.ValidateIsEmpty),
		validation.Field(&c.Status, utils.ValidateIsEmpty),
		validation.Field(&c.StatusMessage, utils.ValidateIsEmpty),
		validation.Field(&c.CreatedAt, utils.ValidateIsEmpty),
		validation.Field(&c.UpdatedAt, utils.ValidateIsEmpty),
		validation.Field(&c.NodeAuthToken, utils.ValidateIsEmpty),
		validation.Field(&c.Image, validation.Required),
	); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

// ValidateResourceRequestFields checks if the Limit fields are valid and can be used to create a new Limit.
func ValidateResourceRequestFields(r *scheduler_proto.ResourceRequest, mask *types.FieldMask) error {
	if r == nil {
		return status.Error(codes.InvalidArgument, "CurrentResourceRequest is nil")
	}
	rules := []*validation.FieldRules{
		validation.Field(&r.Id, utils.ValidateIsEmpty),
		validation.Field(&r.Status, utils.ValidateIsEmpty),
		validation.Field(&r.StatusMessage, utils.ValidateIsEmpty),
		validation.Field(&r.CreatedAt, utils.ValidateIsEmpty),
		validation.Field(&r.UpdatedAt, utils.ValidateIsEmpty),
	}

	if inMask("cpu", mask) {
		rules = append(rules, validation.Field(&r.Cpu, validation.Required))
	}
	if inMask("memory", mask) {
		rules = append(rules, validation.Field(&r.Memory, validation.Required))
	}
	if inMask("disk", mask) {
		rules = append(rules, validation.Field(&r.Disk, validation.Required))
	}

	if err := validation.ValidateStruct(r, rules...); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func inMask(fieldName string, mask *types.FieldMask) bool {
	if mask == nil {
		return true
	}
	for _, path := range mask.Paths {
		if path == fieldName {
			return true
		}
	}
	return false
}

package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// Update updates the user'srv details. Can be executed by admins only.
func (s AccountsServer) Update(ctx context.Context, r *accounts_proto.UpdateUserRequest) (*accounts_proto.User, error) {
	if len(r.GetUpdateMask().GetPaths()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty update_mask field is not allowed")
	}

	user := &models.User{}
	if err := user.LookUp(s.db, r.GetLookup()); err != nil {
		return nil, err
	}

	tx := s.db.Begin()
	event, err := user.Update(tx, r.User, r.UpdateMask)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "user.Update failed"))
	}

	if err := events.CommitAndPublish(tx, s.producer, event); err != nil {
		return nil, errors.Wrap(err, "failed to CommitAndPublish event")
	}

	response, err := user.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create proto response message")
	}

	return response, nil
}

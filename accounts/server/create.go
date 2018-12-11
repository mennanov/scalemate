package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

// Create creates a new user. Can be executed by admins only.
func (s AccountsServer) Create(ctx context.Context, r *accounts_proto.CreateUserRequest) (*accounts_proto.User, error) {
	user := &models.User{}
	if err := user.FromProto(r.GetUser()); err != nil {
		return nil, errors.Wrap(err, "user.FromProto failed")
	}

	if err := user.SetPasswordHash(r.GetPassword(), s.BcryptCost); err != nil {
		return nil, err
	}

	tx := s.DB.Begin()
	event, err := user.Create(tx)
	if err != nil {
		return nil, err
	}

	if err := events.CommitAndPublish(tx, s.Producer, event); err != nil {
		return nil, errors.Wrap(err, "failed to send event and commit")
	}

	response, err := user.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a proto response message")
	}
	return response, nil
}

package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

// Register registers a new user with a "USER" role.
func (s AccountsServer) Register(ctx context.Context, r *accounts_proto.RegisterRequest) (*empty.Empty, error) {
	user := &models.User{
		Username: r.GetUsername(),
		Email:    r.GetEmail(),
		Role:     accounts_proto.User_USER,
		Banned:   false,
	}

	if err := user.SetPasswordHash(r.GetPassword(), s.bCryptCost); err != nil {
		return nil, err
	}

	tx := s.db.Begin()
	event, err := user.Create(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new user")
	}

	if err := events.CommitAndPublish(tx, s.producer, event); err != nil {
		return nil, errors.Wrap(err, "failed to publish event")
	}

	return &empty.Empty{}, nil
}

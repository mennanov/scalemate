package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// Register registers a new User.
func (s AccountsServer) Register(ctx context.Context, r *accounts_proto.RegisterRequest) (*accounts_proto.User, error) {
	user := &models.User{
		Username: r.GetUsername(),
		Email:    r.GetEmail(),
		Banned:   false,
	}

	if err := user.SetPasswordHash(r.GetPassword(), s.bCryptCost); err != nil {
		return nil, err
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start transaction")
	}
	event, err := user.Create(tx)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "failed to create a new user"))
	}

	userProto, err := user.ToProto(nil)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "user.ToProto failed"))
	}

	if err := utils.CommitAndPublish(tx, s.producer, event); err != nil {
		return nil, errors.Wrap(err, "CommitAndPublish failed")
	}

	return userProto, nil
}

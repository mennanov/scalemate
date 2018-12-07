package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// Register registers a new user with a "USER" role.
func (s AccountsServer) Register(ctx context.Context, r *accounts_proto.RegisterRequest) (*empty.Empty, error) {
	user := &models.User{
		Username: r.GetUsername(),
		Email:    r.GetEmail(),
		Role:     accounts_proto.User_USER,
		Banned:   false,
	}

	if err := user.SetPasswordHash(r.GetPassword(), s.BcryptCost); err != nil {
		return nil, err
	}

	tx := s.DB.Begin()
	event, err := user.Create(tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new user")
	}

	publisher, err := utils.NewAMQPPublisher(s.AMQPConnection, utils.AccountsAMQPExchangeName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AMQP publisher instance")
	}
	if err := utils.SendAndCommit(tx, publisher, event); err != nil {
		return nil, errors.Wrap(err, "failed to publish event")
	}

	return &empty.Empty{}, nil
}

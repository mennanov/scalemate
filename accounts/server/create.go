package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// Create creates a new user. Can be executed by admins only.
func (s AccountsServer) Create(ctx context.Context, r *accounts_proto.CreateUserRequest) (*accounts_proto.User, error) {
	user := &models.User{}
	user.FromProto(r.GetUser())

	if err := user.SetPasswordHash(r.GetPassword(), s.BcryptCost); err != nil {
		return nil, err
	}

	tx := s.DB.Begin()
	event, err := user.Create(tx)
	if err != nil {
		return nil, err
	}

	publisher, err := utils.NewAMQPPublisher(s.AMQPConnection, utils.AccountsAMQPExchangeName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AMQP publisher instance")
	}
	if err := utils.SendAndCommit(tx, publisher, event); err != nil {
		return nil, errors.Wrap(err, "failed to send event and commit")
	}

	response, err := user.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a proto response message")
	}
	return response, nil
}

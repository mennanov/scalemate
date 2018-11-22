package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
	"github.com/pkg/errors"
)

// Delete deletes the user from DB. In the current gorm implementation it simply marks the user as deleted but does not
// actually delete the row from the table in DB.
// Can be executed by admins only.
func (s AccountsServer) Delete(ctx context.Context, r *accounts_proto.UserLookupRequest) (*empty.Empty, error) {
	user := &models.User{}
	if err := user.LookUp(s.DB, r); err != nil {
		return nil, err
	}
	tx := s.DB.Begin()
	event, err := user.Delete(tx)
	if err != nil {
		return nil, err
	}

	publisher, err := events.NewAMQPPublisher(s.AMQPConnection, utils.AccountsAMQPExchangeName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AMQP publisher instance")
	}
	if err := utils.SendAndCommit(ctx, tx, publisher, event); err != nil {
		return nil, errors.Wrap(err, "failed to SendAndCommit event")
	}

	return &empty.Empty{}, nil
}

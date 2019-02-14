package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
)

// Delete deletes the user from DB. In the current gorm implementation it simply marks the user as deleted but does not
// actually delete the row from the table in DB.
// Can be executed by admins only.
func (s AccountsServer) Delete(ctx context.Context, r *accounts_proto.UserLookupRequest) (*empty.Empty, error) {
	user := &models.User{}
	if err := user.LookUp(s.db, r); err != nil {
		return nil, err
	}
	tx := s.db.Begin()
	event, err := user.Delete(tx)
	if err != nil {
		return nil, err
	}

	if err := events.CommitAndPublish(tx, s.producer, event); err != nil {
		return nil, errors.Wrap(err, "failed to CommitAndPublish event")
	}

	return &empty.Empty{}, nil
}

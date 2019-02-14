package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
)

// Get gets details for the requested user.
func (s AccountsServer) Get(ctx context.Context, r *accounts_proto.UserLookupRequest) (*accounts_proto.User, error) {
	user := &models.User{}
	if err := user.LookUp(s.db, r); err != nil {
		return nil, err
	}

	response, err := user.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a proto response message")
	}

	return response, nil
}

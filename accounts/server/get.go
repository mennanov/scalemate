package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// Get gets details for the requested user.
func (s AccountsServer) Get(ctx context.Context, r *accounts_proto.UserLookupRequest) (*accounts_proto.User, error) {
	user, err := models.UserLookUp(s.db, r)
	if err != nil {
		return nil, errors.Wrap(err, "UserLookUp failed")
	}
	if err := utils.ClaimsUsernameEqual(ctx, user.Username); err != nil {
		return nil, err
	}

	return &user.User, nil
}


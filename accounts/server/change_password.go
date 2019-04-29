package server

import (
	"context"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/utils"
)

// ChangePassword changes a password for the currently authenticated user.
// Info about the current user is obtained from the JWT claims.
// This method can be executed by admins and the actual user only.
func (s AccountsServer) ChangePassword(
	ctx context.Context,
	r *accounts_proto.ChangePasswordRequest,
) (*types.Empty, error) {
	if err := validation.ValidateStruct(r,
		validation.Field(&r.Username, validation.Required, validation.Match(UsernameRegExp)),
		validation.Field(&r.Password, validation.Required, validation.Length(8, 64)),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := models.UserLookUp(s.db, &accounts_proto.UserLookupRequest{
		Request: &accounts_proto.UserLookupRequest_Username{Username: r.Username}})
	if err != nil {
		return nil, errors.Wrap(err, "UserLookUp failed")
	}
	if err := claimsUsernameEqual(ctx, user.Username); err != nil {
		return nil, err
	}
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start transaction")
	}
	event, err := user.ChangePassword(tx, r.Password, s.bCryptCost)
	if err != nil {
		return nil, utils.RollbackTransaction(tx, errors.Wrap(err, "user.ChangePassword failed"))
	}
	if err := utils.CommitAndPublish(tx, s.producer, event); err != nil {
		return nil, errors.Wrap(err, "CommitAndPublish failed")
	}

	return &types.Empty{}, nil
}

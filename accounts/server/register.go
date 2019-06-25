package server

import (
	"context"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"

	"github.com/go-ozzo/ozzo-validation/is"
)

// UsernameRegExp matches a username for validation.
var UsernameRegExp = regexp.MustCompile(`^[_\w+]{3,32}$`)

// Register registers a new User.
func (s AccountsServer) Register(ctx context.Context, r *accounts_proto.RegisterRequest) (*accounts_proto.User, error) {
	if err := validation.ValidateStruct(r,
		validation.Field(&r.Username, validation.Required, validation.Match(UsernameRegExp)),
		validation.Field(&r.Email, validation.Required, is.Email),
		validation.Field(&r.Password, validation.Required, validation.Length(8, 64)),
	); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user := &models.User{
		User: accounts_proto.User{
			Username: r.Username,
			Email:    r.Email,
			Banned:   false,
		},
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

	if err := events.CommitAndPublish(tx, s.producer, event); err != nil {
		return nil, errors.Wrap(err, "CommitAndPublish failed")
	}

	return &user.User, nil
}

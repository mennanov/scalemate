package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/auth"
)

// Get gets details for the requested user.
func (s AccountsServer) Get(ctx context.Context, r *accounts_proto.UserLookupRequest) (*accounts_proto.User, error) {
	user, err := models.UserLookUp(s.db, r)
	if err != nil {
		return nil, errors.Wrap(err, "UserLookUp failed")
	}
	if err := checkUserPermissions(ctx, user.Username); err != nil {
		return nil, err
	}

	response, err := user.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "user.ToProto failed")
	}

	return response, nil
}

func checkUserPermissions(ctx context.Context, username string) error {
	ctxClaims := ctx.Value(auth.ContextKeyClaims)
	if ctxClaims == nil {
		return status.Error(codes.Unauthenticated, "no JWT claims found")
	}
	claims, ok := ctxClaims.(*auth.Claims)
	if !ok {
		return status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	if claims.Username != username {
		return status.Error(codes.PermissionDenied, "permission denied")
	}
	return nil
}

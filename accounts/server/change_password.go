package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

// ChangePassword changes a password for the currently authenticated user.
// Info about the current user is obtained from the JWT claims.
// This method can be executed by admins and the actual user only.
func (s AccountsServer) ChangePassword(ctx context.Context, r *accounts_proto.ChangePasswordRequest) (*empty.Empty, error) {
	ctxClaims := ctx.Value(auth.ContextKeyClaims)
	if ctxClaims == nil {
		return nil, status.Error(codes.Unauthenticated, "no JWT claims found")
	}
	claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unknown JWT claims type")
	}

	if claims.Username != r.GetUsername() && claims.Role != accounts_proto.User_ADMIN {
		logrus.WithFields(logrus.Fields{
			"claimsUsername": claims.Username,
			"username":       r.Username,
		}).Warn("ChangePassword permission denied")
		return nil, status.Error(
			codes.PermissionDenied, "method 'ChangePassword' is allowed for admins or owners only")
	}

	user := &models.User{}
	if err := user.LookUp(s.DB, &accounts_proto.UserLookupRequest{Username: r.GetUsername()}); err != nil {
		return nil, err
	}
	tx := s.DB.Begin()
	event, err := user.ChangePassword(tx, r.GetPassword(), s.BcryptCost)
	if err != nil {
		return nil, err
	}
	publisher, err := utils.NewAMQPPublisher(s.AMQPConnection, utils.AccountsAMQPExchangeName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AMQP publisher instance")
	}
	if err := utils.SendAndCommit(tx, publisher, event); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

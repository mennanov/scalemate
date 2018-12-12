package server

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
)

// AuthFuncOverride implements an authentication and (partially) authorization for gRPC methods in Accounts Service.
// The context is populated with the claims created from the parsed JWT passed along with the request.
// More: https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#ServiceAuthFuncOverride
func (s AccountsServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	var err error
	switch fullMethodName {
	// Methods available for ADMIN only.
	case "/accounts.accounts_proto.Accounts/Create",
		"/accounts.accounts_proto.Accounts/Delete",
		"/accounts.accounts_proto.Accounts/Update",
		"/accounts.accounts_proto.Accounts/List",
		"/accounts.accounts_proto.Accounts/Get":

		ctx, err = s.ClaimsInjector.InjectClaims(ctx)
		if err != nil {
			return nil, err
		}

		claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
		if !ok {
			return nil, status.Error(codes.Internal, "unknown JWT claims format")
		}

		if claims.Role != accounts_proto.User_ADMIN {
			log := ctxlogrus.Extract(ctx)
			log.WithField("username", claims.Username).
				Warnf("Permission denied for method %s", fullMethodName)
			return nil, status.Errorf(
				codes.PermissionDenied, "method '%s' is allowed for admins only", fullMethodName)
		}

	case "/accounts.accounts_proto.Accounts/ChangePassword":
		// This method is allowed for admins or the user performing the request. This is checked in the handler itself.
		ctx, err = s.ClaimsInjector.InjectClaims(ctx)
		if err != nil {
			return nil, err
		}
	}

	return ctx, nil
}

// AuthFunc is required by the grpc_auth.UnaryServerInterceptor.
// https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#type-authfunc
func AuthFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

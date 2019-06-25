package server

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthFuncOverride implements an authentication and (partially) authorization for gRPC methods in Scheduler Service.
// The context is populated with the claims created from the parsed JWT passed along with the request.
// More: https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#ServiceAuthFuncOverride
func (s SchedulerServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	newCtx, err := s.claimsInjector.InjectClaims(ctx)
	if err != nil {
		st, ok := status.FromError(errors.Cause(err))
		if ok && st.Code() == codes.Unauthenticated {
			// No JWT provided.
			return ctx, nil
		}
		return ctx, err
	}
	return newCtx, nil
}

// AuthFunc is required by the grpc_auth.UnaryServerInterceptor.
// https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#type-authfunc
func AuthFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

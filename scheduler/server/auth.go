package server

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/auth"
)

// AuthFuncOverride implements an authentication and (partially) authorization for gRPC methods in Scheduler Service.
// The context is populated with the claims created from the parsed JWT passed along with the request.
// More: https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#ServiceAuthFuncOverride
func (s SchedulerServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	var err error
	switch fullMethodName {
	case "/scheduler.scheduler_proto.Scheduler/RunJob",
		"/scheduler.scheduler_proto.Scheduler/GetJob",
		"/scheduler.scheduler_proto.Scheduler/ListJobs",
		"/scheduler.scheduler_proto.Scheduler/ReceiveTasks",
		"/scheduler.scheduler_proto.Scheduler/ListTasks",
		"/scheduler.scheduler_proto.Scheduler/GetTask":

		ctx, err = auth.ParseJWTFromContext(ctx, s.JWTSecretKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse JWT")
		}

		//claims, ok := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
		//if !ok {
		//	return nil, status.Error(codes.Internal, "unknown JWT claims format")
		//}
		//
		//if claims.Role != accounts_proto.User_ADMIN {
		//	return nil, status.Errorf(
		//		codes.PermissionDenied, "method '%s' is allowed for admins only", fullMethodName)
		//}

	}

	return ctx, nil
}

// AuthFunc is required by the grpc_auth.UnaryServerInterceptor.
// https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#type-authfunc
func AuthFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

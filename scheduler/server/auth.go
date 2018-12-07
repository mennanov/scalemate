package server

import (
	"context"

	"github.com/pkg/errors"
)

// TODO: create a layer (interface of with the same service method names) for each type of Resource that will handle CRUD operations permissions.
// Example:

// AuthFuncOverride implements an authentication and (partially) authorization for gRPC methods in Scheduler Service.
// The context is populated with the claims created from the parsed JWT passed along with the request.
// More: https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#ServiceAuthFuncOverride
func (s SchedulerServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	var err error
	switch fullMethodName {
	case "/scheduler.scheduler_proto.Scheduler/RunJob",
		"/scheduler.scheduler_proto.Scheduler/CreateJob",
		"/scheduler.scheduler_proto.Scheduler/GetJob",
		"/scheduler.scheduler_proto.Scheduler/ListJobs",
		"/scheduler.scheduler_proto.Scheduler/IterateTasks",
		"/scheduler.scheduler_proto.Scheduler/ReceiveTasksForNode",
		"/scheduler.scheduler_proto.Scheduler/ListTasks",
		"/scheduler.scheduler_proto.Scheduler/GetTask":

		ctx, err = s.ClaimsInjector.InjectClaims(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse JWT")
		}
	}

	return ctx, nil
}

// AuthFunc is required by the grpc_auth.UnaryServerInterceptor.
// https://github.com/grpc-ecosystem/go-grpc-middleware/tree/master/auth#type-authfunc
func AuthFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

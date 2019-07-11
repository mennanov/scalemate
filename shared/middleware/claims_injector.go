package middleware

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
)

// ClaimsInjectorUnaryInterceptor parses the provided claims and injects them to the context.
// If JWT is provided with the request, then the handler is called with the original context (without claims).
// If JWT could not be parsed then handler will not be called and the error will be returned.
func ClaimsInjectorUnaryInterceptor(injector auth.ClaimsInjector) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		newCtx, err := injector.InjectClaims(ctx)
		if err != nil {
			st, ok := status.FromError(errors.Cause(err))
			if ok && st.Code() == codes.Unauthenticated {
				// No JWT provided.
				return handler(ctx, req)
			}
			return nil, err
		}
		return handler(newCtx, req)
	}
}

// ClaimsInjectorStreamingInterceptor is a streaming interceptor similar to ClaimsInjectorUnaryInterceptor, but for
// streaming.
func ClaimsInjectorStreamingInterceptor(injector auth.ClaimsInjector) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		newCtx, err := injector.InjectClaims(ss.Context())
		if err != nil {
			st, ok := status.FromError(errors.Cause(err))
			if ok && st.Code() == codes.Unauthenticated {
				// No JWT provided.
				return handler(srv, ss)
			}
			return err
		}
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
	}
}

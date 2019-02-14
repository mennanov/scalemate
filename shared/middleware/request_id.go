package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// LoggerRequestIDInterceptor adds a field to the logger in the context with a unique request ID string value.
// This is helpful when tracing all the log entries for a single request.
func LoggerRequestIDInterceptor(fieldName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (_ interface{}, err error) {
		ctxlogrus.AddFields(ctx, logrus.Fields{
			fieldName: uuid.New().String(),
		})
		return handler(ctx, req)
	}
}

package middleware_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/sirupsen/logrus"

	"github.com/mennanov/scalemate/shared/middleware"
)

func TestLoggerRequestIDInterceptor(t *testing.T) {
	entry := logrus.NewEntry(logrus.New())
	ctx := ctxlogrus.ToContext(context.Background(), entry)

	fieldName := "request.id"
	interceptor := middleware.LoggerRequestIDInterceptor(fieldName)

	prevRequestID := ""
	for i := 0; i < 3; i++ {
		_, err := interceptor(ctx, struct {
		}{}, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
			logger := ctxlogrus.Extract(ctx)
			requestID, ok := logger.Data[fieldName]
			require.True(t, ok)
			// Verify that the request ID is unique (not equal to the previous one).
			assert.NotEqual(t, prevRequestID, requestID.(string))
			prevRequestID = requestID.(string)
			return nil, nil
		})
		require.NoError(t, err)
	}
}

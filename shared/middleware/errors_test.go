package middleware

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequestsLoggerInterceptor(t *testing.T) {
	logger := logrus.New()
	t.Run("single gRPC error", func(t *testing.T) {
		err := status.Error(codes.PermissionDenied, "root")
		interceptor := RequestsLoggerInterceptor(logger)
		ctx := context.Background()
		_, resErr := interceptor(ctx, nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, err
		})
		assert.Equal(t, err, resErr)
	})

	t.Run("two errors", func(t *testing.T) {
		err := status.Error(codes.PermissionDenied, "root")
		err2 := errors.Wrap(err, "wrapper")
		interceptor := RequestsLoggerInterceptor(logger)
		ctx := context.Background()
		_, resErr := interceptor(ctx, nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, err2
		})
		assert.Equal(t, err, resErr)
	})

	t.Run("three errors", func(t *testing.T) {
		err := errors.New("root")
		err2 := errors.Wrap(err, "wrapper")
		err3 := errors.Wrap(err2, "wrapper2")
		interceptor := RequestsLoggerInterceptor(logger)
		ctx := context.Background()
		_, resErr := interceptor(ctx, nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, err3
		})
		assert.Equal(t, err, resErr)
	})
}

func TestStackAndCause(t *testing.T) {

	t.Run("no errors", func(t *testing.T) {
		s, c := stackAndCause(nil)
		assert.Nil(t, s)
		assert.Nil(t, c)
	})

	t.Run("single error", func(t *testing.T) {
		rootErr := errors.New("root")
		s, c := stackAndCause(rootErr)
		assert.Equal(t, rootErr, s)
		assert.Equal(t, rootErr, c)
	})

	t.Run("two errors with stack", func(t *testing.T) {
		rootErr := errors.New("root")
		wrapperErr := errors.Wrap(rootErr, "wrapper")
		s, c := stackAndCause(wrapperErr)
		assert.Equal(t, rootErr, s)
		assert.Equal(t, rootErr, c)
	})

	t.Run("two errors only one with stack", func(t *testing.T) {
		rootErr := status.Error(codes.PermissionDenied, "root")
		wrapperErr := errors.Wrap(rootErr, "wrapper")
		s, c := stackAndCause(wrapperErr)
		assert.Equal(t, wrapperErr, s)
		assert.Equal(t, rootErr, c)
	})

	t.Run("three errors", func(t *testing.T) {
		rootErr := status.Error(codes.PermissionDenied, "root")
		wrapperErr := errors.Wrap(rootErr, "wrapper")
		wrapperErr2 := errors.Wrap(wrapperErr, "wrapper2")
		s, c := stackAndCause(wrapperErr2)
		assert.Equal(t, wrapperErr, s)
		assert.Equal(t, rootErr, c)
	})
}

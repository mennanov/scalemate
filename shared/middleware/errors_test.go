package middleware

import (
	"context"
	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorsInterceptor(t *testing.T) {
	t.Run("single gRPC error", func(t *testing.T) {
		err := status.Error(codes.PermissionDenied, "root")
		interceptor := StackTraceErrorInterceptor(false, codes.PermissionDenied)
		ctx := context.Background()
		_, resErr := interceptor(ctx, nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, err
		})
		assert.Equal(t, err, resErr)
	})

	t.Run("two error", func(t *testing.T) {
		err := status.Error(codes.PermissionDenied, "root")
		err2 := errors.Wrap(err, "wrapper")
		interceptor := StackTraceErrorInterceptor(false, codes.PermissionDenied)
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
		interceptor := StackTraceErrorInterceptor(false, codes.PermissionDenied)
		ctx := context.Background()
		_, resErr := interceptor(ctx, nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, err3
		})
		s, ok := status.FromError(resErr)
		require.True(t, ok)
		assert.Equal(t, codes.Unknown, s.Code())
		assert.Equal(t, s.Err().Error(), resErr.Error())
	})
}

func TestWrapperAndCause(t *testing.T) {

	t.Run("no errors", func(t *testing.T) {
		w, c := wrapperAndCause(nil)
		assert.Nil(t, w)
		assert.Nil(t, c)
	})

	t.Run("single error", func(t *testing.T) {
		rootErr := errors.New("root")
		w, c := wrapperAndCause(rootErr)
		assert.Nil(t, w)
		assert.Equal(t, c.Error(), rootErr.Error())
	})

	t.Run("two errors", func(t *testing.T) {
		rootErr := errors.New("root")
		wrapperErr := errors.Wrap(rootErr, "wrapper")
		w, c := wrapperAndCause(wrapperErr)
		assert.Equal(t, w.Error(), wrapperErr.Error())
		assert.Equal(t, c.Error(), rootErr.Error())
	})

	t.Run("three errors", func(t *testing.T) {
		rootErr := errors.New("root")
		wrapperErr := errors.Wrap(rootErr, "wrapper")
		wrapperErr2 := errors.Wrap(wrapperErr, "wrapper2")
		w, c := wrapperAndCause(wrapperErr2)
		assert.Equal(t, w.Error(), wrapperErr.Error())
		assert.Equal(t, c.Error(), rootErr.Error())
	})
}

func TestFirstErrorWithStack(t *testing.T) {

	t.Run("no stacks", func(t *testing.T) {
		err := io.EOF
		assert.Nil(t, firstErrorWithStack(err))
	})

	t.Run("one stack", func(t *testing.T) {
		err := io.EOF
		err2 := errors.WithStack(err)
		stackErr := firstErrorWithStack(err, err2)
		assert.NotNil(t, stackErr.(error).Error())
	})
}

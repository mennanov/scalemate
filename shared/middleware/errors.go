package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type causer interface {
	Cause() error
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// stackAndCause returns a stackTracer from the first (deepest) error that provides it and the cause error.
func stackAndCause(err error) (stackTracer, error) {
	var tracer stackTracer
	for ; ; {
		t, ok := err.(stackTracer)
		if ok {
			tracer = t
		}
		errCause, ok := err.(causer)
		if ok {
			err = errCause.Cause()
		} else {
			break
		}
	}
	return tracer, err
}

type contextKey struct {
	value string
}

func (c *contextKey) String() string {
	return c.value
}

// requestLoggerContextKey is the key used to store a requests logger in a context.
var requestLoggerContextKey = &contextKey{"logger"}

// RequestsLoggerInterceptor adds a field to the logger with a unique request ID string value.
func RequestsLoggerInterceptor(logger *logrus.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (_ interface{}, err error) {
		start := time.Now()
		entry := logger.WithField("grpc.request_id", uuid.New().String())
		if info != nil {
			entry = entry.WithField("grpc.method", info.FullMethod)
		}
		newCtx := context.WithValue(ctx, requestLoggerContextKey, entry)
		response, err := handler(newCtx, req)
		if err != nil {
			entry = entry.WithField("error", err.Error())
			errWithStack, errCause := stackAndCause(err)
			if errWithStack != nil {
				entry = entry.WithField("stacktrace", fmt.Sprintf("%+v", errWithStack.StackTrace()))
			}
			s, ok := status.FromError(errCause)
			if ok {
				entry = entry.WithField("grpc.code", s.Code().String())
			}
			err = errCause
		}

		entry.WithField("grpc.duration_ns", time.Now().Sub(start).Nanoseconds()).Info("request finished")
		return response, err
	}
}

// GetCtxLogger gets the requests logger from the given context.
// Will panic if there is no request logger in the context.
func GetCtxLogger(ctx context.Context) *logrus.Logger {
	return ctx.Value(requestLoggerContextKey).(*logrus.Logger)
}

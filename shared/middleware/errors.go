package middleware

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// wrapperAndCause returns the root error and the immediate wrapper of it.
// Both returned errors can be nil.
func wrapperAndCause(err error) (error, error) { //revive:disable-line:error-return
	type causer interface {
		Cause() error
	}

	if err == nil {
		return nil, nil
	}

	var e error
	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			return e, err
		}
		e = err
		err = cause.Cause()
	}
	return e, err
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// firstErrorWithStack returns a stackTracer from the first error that provides it.
func firstErrorWithStack(errors ...error) stackTracer {
	for _, err := range errors {
		stackErr, ok := err.(stackTracer)
		if ok {
			return stackErr
		}
	}
	return nil
}

// StackTraceErrorInterceptor unwraps the error, logs the stack trace if the error's code is listed in the logCodes.
// If `logAll` is true, the error will be logged even if there is no stack trace available.
func StackTraceErrorInterceptor(logAll bool, logCodes ...codes.Code) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (_ interface{}, err error) {
		response, err := handler(ctx, req)
		if err != nil {
			wrapperErr, rootErr := wrapperAndCause(err)
			errorStatus, _ := status.FromError(rootErr)
			for _, code := range logCodes {
				if errorStatus.Code() == code {
					entry := ctxlogrus.Extract(ctx)
					// Try to log a stack trace if it's available.
					stackErr := firstErrorWithStack(rootErr, wrapperErr)
					if stackErr != nil {
						// Log the first 5 stack trace frames.
						entry.WithField("stacktrace", fmt.Sprintf("%+v", stackErr.StackTrace()[:5])).
							Errorf("%s", err.Error())
					} else if logAll {
						entry.Errorf("%s", err.Error())
					}
					break
				}
			}
			return response, errorStatus.Err()
		}
		return response, err
	}
}

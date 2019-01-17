package client

import (
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrMsgFmt returns an error string for the given gRPC status.
type ErrMsgFmt func(s *status.Status) string

// GRPCErrorMessages is a set of functions to be used to print out an error string.
type GRPCErrorMessages struct {
	OK                 ErrMsgFmt
	Canceled           ErrMsgFmt
	Unknown            ErrMsgFmt
	InvalidArgument    ErrMsgFmt
	DeadlineExceeded   ErrMsgFmt
	NotFound           ErrMsgFmt
	AlreadyExists      ErrMsgFmt
	PermissionDenied   ErrMsgFmt
	ResourceExhausted  ErrMsgFmt
	FailedPrecondition ErrMsgFmt
	Aborted            ErrMsgFmt
	OutOfRange         ErrMsgFmt
	Unimplemented      ErrMsgFmt
	Internal           ErrMsgFmt
	Unavailable        ErrMsgFmt
	DataLoss           ErrMsgFmt
	Unauthenticated    ErrMsgFmt
}

// ErrorView handles error messages representation.
func ErrorView(errWriter io.Writer, errMessages *GRPCErrorMessages, err error) {
	msg := err.Error()
	if statusCode, ok := status.FromError(err); ok {
		switch statusCode.Code() {
		case codes.OK:
			if errMessages.OK != nil {
				msg = errMessages.OK(statusCode)
			}
		case codes.Canceled:
			if errMessages.Canceled != nil {
				msg = errMessages.Canceled(statusCode)
			}
		case codes.InvalidArgument:
			if errMessages.InvalidArgument != nil {
				msg = errMessages.InvalidArgument(statusCode)
			}
		case codes.DeadlineExceeded:
			if errMessages.DeadlineExceeded != nil {
				msg = errMessages.DeadlineExceeded(statusCode)
			}
		case codes.NotFound:
			if errMessages.NotFound != nil {
				msg = errMessages.NotFound(statusCode)
			}
		case codes.AlreadyExists:
			if errMessages.AlreadyExists != nil {
				msg = errMessages.AlreadyExists(statusCode)
			}
		case codes.PermissionDenied:
			if errMessages.PermissionDenied != nil {
				msg = errMessages.PermissionDenied(statusCode)
			}
		case codes.ResourceExhausted:
			if errMessages.ResourceExhausted != nil {
				msg = errMessages.ResourceExhausted(statusCode)
			}
		case codes.FailedPrecondition:
			if errMessages.FailedPrecondition != nil {
				msg = errMessages.FailedPrecondition(statusCode)
			}
		case codes.Aborted:
			if errMessages.Aborted != nil {
				msg = errMessages.Aborted(statusCode)
			}
		case codes.OutOfRange:
			if errMessages.OutOfRange != nil {
				msg = errMessages.OutOfRange(statusCode)
			}
		case codes.Unimplemented:
			if errMessages.Unimplemented != nil {
				msg = errMessages.Unimplemented(statusCode)
			}
		case codes.Internal:
			if errMessages.Internal != nil {
				msg = errMessages.Internal(statusCode)
			}
		case codes.Unavailable:
			if errMessages.Unavailable != nil {
				msg = errMessages.Unavailable(statusCode)
			}
		case codes.DataLoss:
			if errMessages.DataLoss != nil {
				msg = errMessages.DataLoss(statusCode)
			}
		case codes.Unauthenticated:
			if errMessages.Unauthenticated != nil {
				msg = errMessages.Unauthenticated(statusCode)
			}
		}
	}
	DisplayMessage(errWriter, msg)
}

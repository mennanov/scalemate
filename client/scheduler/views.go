package scheduler

import (
	"fmt"
	"io"

	"github.com/golang/protobuf/jsonpb"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/client"
)

// JSONIndent is used as an indent for marshalling protobufs as JSON.
const JSONIndent = "  "

// CreateJobView handles CreateJobController results representation.
func CreateJobView(logger *logrus.Logger, jsonOut io.Writer, job *scheduler_proto.Job, err error) {
	if err != nil {
		client.ErrorView(logger, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid parameters: %s", s.Message())
			},
			PermissionDenied: func(s *status.Status) string {
				return fmt.Sprintf("permission denied: %s", s.Message())
			},
			Unauthenticated: func(s *status.Status) string {
				return fmt.Sprintf("unauthenticated: %s", s.Message())
			},
		}, err)
		return
	}
	logger.Info("Job has been created.")
	m := &jsonpb.Marshaler{
		Indent: JSONIndent,
	}
	if err := m.Marshal(jsonOut, job); err != nil {
		logger.WithError(err).Error("failed to marshal Job proto to JSON")
	}
}

// GetJobView handles GetJobController results representation.
func GetJobView(logger *logrus.Logger, jsonOut io.Writer, job *scheduler_proto.Job, err error) {
	if err != nil {
		client.ErrorView(logger, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid parameters: %s", s.Message())
			},
			PermissionDenied: func(s *status.Status) string {
				return fmt.Sprintf("permission denied: %s", s.Message())
			},
			Unauthenticated: func(s *status.Status) string {
				return fmt.Sprintf("unauthenticated: %s", s.Message())
			},
		}, err)
		return
	}
	m := &jsonpb.Marshaler{
		Indent: "  ",
	}
	if err := m.Marshal(jsonOut, job); err != nil {
		logger.WithError(err).Error("failed to marshal Job proto to JSON")
	}
}

// ListJobsView handles ListJobsController results representation.
func ListJobsView(logger *logrus.Logger, jsonOut io.Writer, response *scheduler_proto.ListJobsResponse, err error) {
	if err != nil {
		client.ErrorView(logger, &client.GRPCErrorMessages{
			InvalidArgument: func(s *status.Status) string {
				return fmt.Sprintf("invalid parameters: %s", s.Message())
			},
			PermissionDenied: func(s *status.Status) string {
				return fmt.Sprintf("permission denied: %s", s.Message())
			},
			Unauthenticated: func(s *status.Status) string {
				return fmt.Sprintf("unauthenticated: %s", s.Message())
			},
		}, err)
		return
	}
	m := &jsonpb.Marshaler{
		Indent: JSONIndent,
	}
	if err := m.Marshal(jsonOut, response); err != nil {
		logger.WithError(err).Error("failed to marshal ListJobsResponse proto to JSON")
	}
}
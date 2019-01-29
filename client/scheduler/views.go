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

// CreateJobView handles ChangePasswordController results representation.
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
	m := &jsonpb.Marshaler{}
	if err := m.Marshal(jsonOut, job); err != nil {
		logger.WithError(err).Error("failed to marshal Job proto to JSON")
	}
}

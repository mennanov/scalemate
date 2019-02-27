package scheduler

import (
	"fmt"
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/client"
)

// JSONIndent is used as an indent for marshalling protobufs as JSON.
const JSONIndent = "  "

// JSONPbView writes a JSON representation of the provided proto message to `jsonOut` and logs errors if any.
func JSONPbView(logger *logrus.Logger, jsonOut io.Writer, pb proto.Message, err error) {
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
	if err := m.Marshal(jsonOut, pb); err != nil {
		logger.WithError(err).Error("failed to marshal protobuf message to JSON")
	}
}

package scheduler

import (
	"context"

	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
)

var (
	loadTokensFailedErrMsg       = "failed to load authentication tokens. Are you logged in?"
	newClaimsFromStringFailedMsg = "failed to parse authentication tokens. Try re-login"
)

// GetNodeController gets an existing Node by its ID.
func GetNodeController(
	schedulerClient scheduler_proto.SchedulerClient,
	nodeID int64,
) (*scheduler_proto.Node, error) {
	return schedulerClient.GetNode(
		context.Background(),
		&scheduler_proto.NodeLookupRequest{NodeId: nodeID})
}

// ListNodesController lists the Nodes that satisfy the criteria.
func ListNodesController(
	schedulerClient scheduler_proto.SchedulerClient,
	flags *ListNodesCmdFlags,
) (*scheduler_proto.ListNodesResponse, error) {
	r := flags.ToProto()
	return schedulerClient.ListNodes(context.Background(), r)
}

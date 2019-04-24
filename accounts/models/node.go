package models

import (
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"

	"github.com/mennanov/scalemate/shared/utils"
)

// Node defines a Node model in DB.
// This model is a subset of the scheduler.models.Node model and holds a minimum information that is required to
// authenticate a Node instead of making a gRPC query to the Scheduler service.
// The data in this DB table is populated by listening to the Scheduler service events when a new Node is created.
type Node struct {
	utils.Model
	Username    string
	Name        string
	Fingerprint []byte
}

// NewNodeFromSchedulerProto creates a new Node instance from `scheduler_proto.Node`.
func NewNodeFromSchedulerProto(p *scheduler_proto.Node) *Node {
	return &Node{
		Model: utils.Model{
			ID: p.Id,
		},
		Username:    p.Username,
		Name:        p.Name,
		Fingerprint: p.Fingerprint,
	}
}

// Create creates a new Node in DB.
func (n *Node) Create(db utils.SqlxGetter) error {
	return utils.HandleDBError(db.Get(n,
		"INSERT INTO nodes (id, username, name, fingerprint) VALUES ($1, $2, $3, $4) RETURNING *",
		n.ID, n.Username, n.Name, n.Fingerprint))
}

// NodeLookUp gets the Node from DB by a username and a Node name.
func NodeLookUp(db utils.SqlxGetter, username, name string) (*Node, error) {
	node := &Node{}
	if err := utils.HandleDBError(
		db.Get(node, "SELECT * FROM nodes WHERE username = $1 AND name = $2", username, name)); err != nil {
		return nil, errors.Wrap(err, "failed to find Node in DB")
	}
	return node, nil
}

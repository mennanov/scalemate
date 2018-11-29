package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"

	"github.com/mennanov/scalemate/accounts/models"
)

// List finds users that satisfy the given constraints in `ListUsersRequest`.
func (s AccountsServer) List(ctx context.Context, r *accounts_proto.ListUsersRequest) (*accounts_proto.ListUsersResponse, error) {
	var users models.Users
	totalCount, err := users.List(s.DB, r)
	if err != nil {
		return nil, err
	}
	response := &accounts_proto.ListUsersResponse{TotalCount: totalCount}

	// Populate response with users.
	for _, user := range users {
		u, err := user.ToProto(nil)
		if err != nil {
			return nil, err
		}
		response.User = append(response.User, u)
	}

	return response, nil
}

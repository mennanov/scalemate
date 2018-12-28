package server

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/accounts/models"
)

// PasswordNodeAuth authenticates a Node by a username, password, Node name and hardware specs.
// It creates a JWT to be used later for other methods across all the other services. This JWT will contain the name of
// the Node.
func (s AccountsServer) PasswordNodeAuth(
	ctx context.Context,
	r *accounts_proto.PasswordNodeAuthRequest,
) (*accounts_proto.AuthTokens, error) {
	user := &models.User{}
	if err := user.LookUp(s.DB, &accounts_proto.UserLookupRequest{Username: r.GetUsername()}); err != nil {
		return nil, err
	}
	if err := user.ComparePassword(r.GetPassword()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "incorrect password")
	}
	// Perform this check after the password is verified to avoid banned users scans.
	if err := user.IsAllowedToAuthenticate(); err != nil {
		return nil, err
	}
	node := &models.Node{}
	if err := node.Get(s.DB, r.GetUsername(), r.GetNodeName()); err != nil {
		return nil, err
	}
	// Perform hardware models checks.
	if r.GetCpuModel() != node.CpuModel {
		return nil, status.Errorf(
			codes.InvalidArgument, "CPU model '%s' does not match the existing record '%s'",
			r.GetCpuModel(), node.CpuModel)
	}
	if r.GetGpuModel() != node.GpuModel {
		return nil, status.Errorf(codes.InvalidArgument,
			"GPU model '%s' does not match the existing record '%s'", r.GetGpuModel(), node.GpuModel)
	}
	if r.GetDiskModel() != node.DiskModel {
		return nil, status.Errorf(codes.InvalidArgument,
			"disk model '%s' does not match the existing record '%s'", r.GetDiskModel(), node.DiskModel)
	}
	if r.GetMemoryModel() != node.MemoryModel {
		return nil, status.Errorf(codes.InvalidArgument,
			"memory model '%s' does not match the existing record '%s'", r.GetMemoryModel(), node.MemoryModel)
	}

	// Generate auth tokens.
	response, err := user.GenerateAuthTokensResponse(
		s.AccessTokenTTL, s.RefreshTokenTTL, s.JWTSecretKey, r.GetNodeName())
	if err != nil {
		return nil, err
	}

	return response, nil
}

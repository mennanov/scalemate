package server_test

import (
	"context"
	"fmt"
	"time"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/accounts/models"
	"google.golang.org/grpc/codes"
)

func (s *ServerTestSuite) TestPasswordNodeAuth() {
	user := s.createTestUserQuick("password")
	node := &models.Node{
		Username:    user.Username,
		Name:        "node_name",
		CpuModel:    "Intel Core i7 @ 2.20GHz",
		GpuModel:    "Intel Iris Pro 1536MB",
		MemoryModel: "DDR3-1600MHz",
		DiskModel:   "251GB APPLE SSD SM0256F",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	ctx := context.Background()
	req := &accounts_proto.PasswordNodeAuthRequest{
		Username:    user.Username,
		Password:    "password",
		NodeName:    "node_name",
		CpuModel:    "Intel Core i7 @ 2.20GHz",
		GpuModel:    "Intel Iris Pro 1536MB",
		MemoryModel: "DDR3-1600MHz",
		DiskModel:   "251GB APPLE SSD SM0256F",
	}

	res, err := s.client.PasswordNodeAuth(ctx, req)
	fmt.Println("response:", res)
	s.Require().NoError(err)

	s.assertAuthTokensValid(res, user, time.Now(), node.Name)
}

func (s *ServerTestSuite) TestPasswordNodeAuth_NodeNotFound() {
	user := s.createTestUserQuick("password")
	node := &models.Node{
		Username: user.Username,
		Name:     "node_name",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	ctx := context.Background()
	req := &accounts_proto.PasswordNodeAuthRequest{
		Username: user.Username,
		Password: "password",
		NodeName: "incorrect_node_name",
	}

	res, err := s.client.PasswordNodeAuth(ctx, req)
	s.assertGRPCError(err, codes.NotFound)
	s.Nil(res)
}

func (s *ServerTestSuite) TestPasswordNodeAuth_HardwareModelsChecksFail() {
	user := s.createTestUserQuick("password")
	node := &models.Node{
		Username:    user.Username,
		Name:        "node_name",
		CpuModel:    "Intel Core i7 @ 2.20GHz",
		GpuModel:    "Intel Iris Pro 1536MB",
		MemoryModel: "DDR3-1600MHz",
		DiskModel:   "251GB APPLE SSD SM0256F",
	}
	_, err := node.Create(s.service.DB)
	s.Require().NoError(err)

	ctx := context.Background()
	requests := []*accounts_proto.PasswordNodeAuthRequest{
		{
			Username: user.Username,
			Password: "password",
			NodeName: "node_name",
			// Incorrect CPU model.
			CpuModel:    "Intel Core i5 @ 2.20GHz",
			GpuModel:    "Intel Iris Pro 1536MB",
			MemoryModel: "DDR3-1600MHz",
			DiskModel:   "251GB APPLE SSD SM0256F",
		},
		{
			Username: user.Username,
			Password: "password",
			NodeName: "node_name",
			CpuModel: "Intel Core i7 @ 2.20GHz",
			// Incorrect GPU model.
			GpuModel:    "Intel Iris Pro 2000MB",
			MemoryModel: "DDR3-1600MHz",
			DiskModel:   "251GB APPLE SSD SM0256F",
		},
		{
			Username:    user.Username,
			Password:    "password",
			NodeName:    "node_name",
			CpuModel:    "Intel Core i7 @ 2.20GHz",
			GpuModel:    "Intel Iris Pro 1536MB",
			MemoryModel: "DDR3-1600MHz",
			// Incorrect disk model.
			DiskModel: "200GB APPLE SSD SM0256F",
		},
		{
			Username: user.Username,
			Password: "password",
			NodeName: "node_name",
			CpuModel: "Intel Core i7 @ 2.20GHz",
			GpuModel: "Intel Iris Pro 1536MB",
			// Incorrect memory model.
			MemoryModel: "DDR5-1600MHz",
			DiskModel:   "251GB APPLE SSD SM0256F",
		},
	}
	for _, req := range requests {
		res, err := s.client.PasswordNodeAuth(ctx, req)
		s.assertGRPCError(err, codes.InvalidArgument)
		s.Nil(res)
	}
}

func (s *ServerTestSuite) TestPasswordNodeAuth_IncorrectPassword() {
	user := s.createTestUserQuick("password")

	ctx := context.Background()
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "incorrect password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.assertGRPCError(err, codes.InvalidArgument)
	s.Nil(res)
}

func (s *ServerTestSuite) TestPasswordNodeAuth_BannedUser() {
	user := s.createTestUser(&models.User{
		Username: "username",
		Banned:   true,
	}, "password")

	ctx := context.Background()
	req := &accounts_proto.PasswordAuthRequest{
		Username: user.Username,
		Password: "password",
	}

	res, err := s.client.PasswordAuth(ctx, req)
	s.assertGRPCError(err, codes.PermissionDenied)
	s.Nil(res)
}

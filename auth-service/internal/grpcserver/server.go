package grpcserver

import (
	"context"

	authv1 "github.com/microservices/common/auth/v1"
	"github.com/microservices/auth-service/internal/client"
	"github.com/microservices/auth-service/internal/keys"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	keyManager *keys.KeyManager
	dbClient   *client.DBClient
}

func NewServer(km *keys.KeyManager, dbClient *client.DBClient) *Server {
	return &Server{
		keyManager: km,
		dbClient:   dbClient,
	}
}

func (s *Server) GetJWKS(ctx context.Context, req *authv1.GetJWKSRequest) (*authv1.GetJWKSResponse, error) {
	return &authv1.GetJWKSResponse{
		JwksJson: s.keyManager.GetJWKSJSON(),
	}, nil
}

func (s *Server) GetUserProfile(ctx context.Context, req *authv1.GetUserProfileRequest) (*authv1.GetUserProfileResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	userResp, err := s.dbClient.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user profile not found: %v", err)
	}

	return &authv1.GetUserProfileResponse{
		UserId: userResp.GetId(),
		Email:  userResp.GetEmail(),
		Name:   userResp.GetName(),
		Role:   userResp.GetRole(),
	}, nil
}

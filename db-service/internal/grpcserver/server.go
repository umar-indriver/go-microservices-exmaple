package grpcserver

import (
	"context"
	"errors"
	"time"

	dbv1 "github.com/microservices/common/db/v1"
	"github.com/microservices/db-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	dbv1.UnimplementedDbServiceServer
	repo repository.UserRepository
}

func NewServer(repo repository.UserRepository) *Server {
	return &Server{repo: repo}
}

func (s *Server) CreateUser(ctx context.Context, req *dbv1.CreateUserRequest) (*dbv1.CreateUserResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" || req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "email, password, and name are required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	user, err := s.repo.CreateUser(ctx, req.GetEmail(), string(hashedPassword), req.GetName(), req.GetRole())
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &dbv1.CreateUserResponse{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) GetUserByEmail(ctx context.Context, req *dbv1.GetUserByEmailRequest) (*dbv1.GetUserByEmailResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	user, err := s.repo.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &dbv1.GetUserByEmailResponse{
		Id:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
		Role:         user.Role,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) GetUserByID(ctx context.Context, req *dbv1.GetUserByIDRequest) (*dbv1.GetUserByIDResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	user, err := s.repo.GetUserByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return &dbv1.GetUserByIDResponse{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) SaveRefreshToken(ctx context.Context, req *dbv1.SaveRefreshTokenRequest) (*dbv1.SaveRefreshTokenResponse, error) {
	if req.GetUserId() == "" || req.GetTokenHash() == "" || req.GetExpiresAt() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, token_hash, and expires_at are required")
	}

	expTime, err := time.Parse(time.RFC3339, req.GetExpiresAt())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid expires_at format, expected RFC3339: %v", err)
	}

	rt, err := s.repo.SaveRefreshToken(ctx, req.GetUserId(), req.GetTokenHash(), expTime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save refresh token: %v", err)
	}

	return &dbv1.SaveRefreshTokenResponse{
		Id: rt.ID,
	}, nil
}

func (s *Server) GetRefreshToken(ctx context.Context, req *dbv1.GetRefreshTokenRequest) (*dbv1.GetRefreshTokenResponse, error) {
	if req.GetTokenHash() == "" {
		return nil, status.Error(codes.InvalidArgument, "token_hash is required")
	}

	rt, err := s.repo.GetRefreshToken(ctx, req.GetTokenHash())
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return nil, status.Error(codes.NotFound, "refresh token not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get refresh token: %v", err)
	}

	return &dbv1.GetRefreshTokenResponse{
		Id:        rt.ID,
		UserId:    rt.UserID,
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt.Format(time.RFC3339),
		Revoked:   rt.Revoked,
	}, nil
}

func (s *Server) RevokeRefreshToken(ctx context.Context, req *dbv1.RevokeRefreshTokenRequest) (*dbv1.RevokeRefreshTokenResponse, error) {
	if req.GetTokenHash() == "" && req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "token_hash or user_id is required")
	}

	err := s.repo.RevokeRefreshToken(ctx, req.GetTokenHash(), req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to revoke refresh token: %v", err)
	}

	return &dbv1.RevokeRefreshTokenResponse{
		Success: true,
	}, nil
}

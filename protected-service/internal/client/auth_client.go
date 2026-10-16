package client

import (
	"context"
	"fmt"
	"log"
	"time"

	authv1 "github.com/microservices/common/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	client authv1.AuthServiceClient
	conn   *grpc.ClientConn
}

func NewAuthClient(targetAddr string) (*AuthClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		targetAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth-service via gRPC at %s: %w", targetAddr, err)
	}

	log.Printf("[Protected-Service] Connected to Auth-Service gRPC server at %s", targetAddr)
	return &AuthClient{
		client: authv1.NewAuthServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *AuthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *AuthClient) GetUserProfile(ctx context.Context, userID string) (*authv1.GetUserProfileResponse, error) {
	return c.client.GetUserProfile(ctx, &authv1.GetUserProfileRequest{
		UserId: userID,
	})
}

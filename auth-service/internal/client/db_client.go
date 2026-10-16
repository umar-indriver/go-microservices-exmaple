package client

import (
	"context"
	"fmt"
	"log"
	"time"

	dbv1 "github.com/microservices/common/db/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DBClient struct {
	client dbv1.DbServiceClient
	conn   *grpc.ClientConn
}

func NewDBClient(targetAddr string) (*DBClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		targetAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db-service at %s: %w", targetAddr, err)
	}

	log.Printf("[Auth-Service] Connected to DB-Service via gRPC at %s", targetAddr)
	return &DBClient{
		client: dbv1.NewDbServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *DBClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *DBClient) CreateUser(ctx context.Context, email, password, name, role string) (*dbv1.CreateUserResponse, error) {
	return c.client.CreateUser(ctx, &dbv1.CreateUserRequest{
		Email:    email,
		Password: password,
		Name:     name,
		Role:     role,
	})
}

func (c *DBClient) GetUserByEmail(ctx context.Context, email string) (*dbv1.GetUserByEmailResponse, error) {
	return c.client.GetUserByEmail(ctx, &dbv1.GetUserByEmailRequest{
		Email: email,
	})
}

func (c *DBClient) GetUserByID(ctx context.Context, id string) (*dbv1.GetUserByIDResponse, error) {
	return c.client.GetUserByID(ctx, &dbv1.GetUserByIDRequest{
		Id: id,
	})
}

func (c *DBClient) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*dbv1.SaveRefreshTokenResponse, error) {
	return c.client.SaveRefreshToken(ctx, &dbv1.SaveRefreshTokenRequest{
		UserId:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

func (c *DBClient) GetRefreshToken(ctx context.Context, tokenHash string) (*dbv1.GetRefreshTokenResponse, error) {
	return c.client.GetRefreshToken(ctx, &dbv1.GetRefreshTokenRequest{
		TokenHash: tokenHash,
	})
}

func (c *DBClient) RevokeRefreshToken(ctx context.Context, tokenHash, userID string) error {
	_, err := c.client.RevokeRefreshToken(ctx, &dbv1.RevokeRefreshTokenRequest{
		TokenHash: tokenHash,
		UserId:    userID,
	})
	return err
}

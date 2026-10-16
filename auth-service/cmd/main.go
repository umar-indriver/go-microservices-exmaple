package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authv1 "github.com/microservices/common/auth/v1"
	"github.com/microservices/auth-service/internal/client"
	"github.com/microservices/auth-service/internal/config"
	"github.com/microservices/auth-service/internal/grpcserver"
	"github.com/microservices/auth-service/internal/httphandler"
	"github.com/microservices/auth-service/internal/jwt"
	"github.com/microservices/auth-service/internal/keys"
	"github.com/microservices/auth-service/internal/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Println("[Auth-Service] Starting Ed25519 Dual-Token Authentication Microservice (YAML Configured)...")

	cfg := config.LoadConfig()

	keyManager, err := keys.NewKeyManager()
	if err != nil {
		log.Fatalf("[Auth-Service] Failed to initialize Ed25519 key manager: %v", err)
	}

	tokenService := jwt.NewTokenService(keyManager, cfg.JWTIssuer(), cfg.AccessExpirationMinutes(), cfg.RefreshExpirationDays())

	dbClient, err := client.NewDBClient(cfg.DBServiceAddr())
	if err != nil {
		log.Printf("[Auth-Service] WARNING: Could not establish immediate gRPC connection to DB-Service at %s: %v", cfg.DBServiceAddr(), err)
	} else {
		defer dbClient.Close()
	}

	redisClient, err := redis.NewRedisClient(cfg.RedisAddr())
	if err != nil {
		log.Printf("[Auth-Service] WARNING: Could not connect to Redis at %s: %v", cfg.RedisAddr(), err)
	} else {
		defer redisClient.Close()
	}

	mux := http.NewServeMux()
	httpHandler := httphandler.NewHTTPHandler(dbClient, tokenService, keyManager, redisClient)
	httpHandler.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    cfg.HTTPPort(),
		Handler: mux,
	}

	go func() {
		log.Printf("[Auth-Service] HTTP REST server listening on %s", cfg.HTTPPort())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Auth-Service] HTTP server error: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", cfg.GRPCPort())
	if err != nil {
		log.Fatalf("[Auth-Service] Failed to listen on gRPC port %s: %v", cfg.GRPCPort(), err)
	}

	grpcServer := grpc.NewServer()
	grpcSrvImpl := grpcserver.NewServer(keyManager, dbClient)
	authv1.RegisterAuthServiceServer(grpcServer, grpcSrvImpl)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("[Auth-Service] gRPC server listening on %s", cfg.GRPCPort())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[Auth-Service] gRPC server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[Auth-Service] Shutting down servers gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("[Auth-Service] HTTP shutdown error: %v", err)
	}
	grpcServer.GracefulStop()

	log.Println("[Auth-Service] Service stopped successfully.")
}

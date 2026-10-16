package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/microservices/protected-service/internal/client"
	"github.com/microservices/protected-service/internal/config"
	"github.com/microservices/protected-service/internal/httphandler"
	"github.com/microservices/protected-service/internal/jwks"
	"github.com/microservices/protected-service/internal/middleware"
	"github.com/microservices/protected-service/internal/redis"
)

func main() {
	log.Println("[Protected-Service] Starting Sample Protected Microservice (YAML Configured)...")

	cfg := config.LoadConfig()

	redisClient, err := redis.NewRedisClient(cfg.RedisAddr())
	if err != nil {
		log.Printf("[Protected-Service] WARNING: Could not connect to Redis at %s: %v", cfg.RedisAddr(), err)
	} else {
		defer redisClient.Close()
	}

	jwksCache := jwks.NewJWKSCache(cfg.AuthHTTPURL(), 15*time.Minute)

	if err := jwksCache.RefreshKeys(); err != nil {
		log.Printf("[Protected-Service] WARNING: Could not pre-fetch JWKS from %s: %v", cfg.AuthHTTPURL(), err)
	} else {
		log.Printf("[Protected-Service] Pre-fetched JWKS public keys successfully from %s", cfg.AuthHTTPURL())
	}

	authClient, err := client.NewAuthClient(cfg.AuthGRPCAddr())
	if err != nil {
		log.Printf("[Protected-Service] WARNING: Could not establish immediate gRPC connection to Auth-Service at %s: %v", cfg.AuthGRPCAddr(), err)
	} else {
		defer authClient.Close()
	}

	authMiddleware := middleware.NewAuthMiddleware(jwksCache, redisClient)
	protectedHandler := httphandler.NewProtectedHandler(authClient)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/protected/data", authMiddleware.Authenticate(http.HandlerFunc(protectedHandler.HandleProtectedData)))

	httpServer := &http.Server{
		Addr:    cfg.HTTPPort(),
		Handler: mux,
	}

	go func() {
		log.Printf("[Protected-Service] HTTP REST server listening on %s", cfg.HTTPPort())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Protected-Service] HTTP server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[Protected-Service] Shutting down HTTP server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("[Protected-Service] HTTP shutdown error: %v", err)
	}

	log.Println("[Protected-Service] Service stopped successfully.")
}

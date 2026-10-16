package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	dbv1 "github.com/microservices/common/db/v1"
	"github.com/microservices/db-service/internal/config"
	"github.com/microservices/db-service/internal/db"
	"github.com/microservices/db-service/internal/grpcserver"
	"github.com/microservices/db-service/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Println("[DB-Service] Starting Database Microservice (YAML Configured)...")

	cfg := config.LoadConfig()

	sqlDB, err := db.InitDB(cfg)
	if err != nil {
		log.Fatalf("[DB-Service] Failed to initialize PostgreSQL database: %v", err)
	}
	defer sqlDB.Close()

	userRepo := repository.NewUserRepository(sqlDB)
	grpcSrvImpl := grpcserver.NewServer(userRepo)

	lis, err := net.Listen("tcp", cfg.GRPCPort())
	if err != nil {
		log.Fatalf("[DB-Service] Failed to listen on gRPC port %s: %v", cfg.GRPCPort(), err)
	}

	grpcServer := grpc.NewServer()
	dbv1.RegisterDbServiceServer(grpcServer, grpcSrvImpl)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("[DB-Service] gRPC server listening on %s", cfg.GRPCPort())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[DB-Service] gRPC server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[DB-Service] Shutting down gRPC server gracefully...")
	grpcServer.GracefulStop()
	log.Println("[DB-Service] Server stopped successfully.")
}

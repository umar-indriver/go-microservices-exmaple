#!/bin/bash

echo "=== Stopping Microservices Processes & Docker Containers ==="

# 1. Kill any running Go microservice binaries
echo "[Process] Stopping Go microservices (db-service, auth-service, protected-service)..."
pkill -f "go run cmd/main.go" || true
pkill -f "db-service" || true
pkill -f "auth-service" || true
pkill -f "protected-service" || true

# 2. Stop Docker Containers
echo "[Docker] Stopping PostgreSQL container 'microservices-postgres'..."
docker stop microservices-postgres 2>/dev/null || true
docker rm microservices-postgres 2>/dev/null || true

echo "[Docker] Stopping Redis container 'microservices-redis'..."
docker stop microservices-redis 2>/dev/null || true
docker rm microservices-redis 2>/dev/null || true

echo "=== All Microservices and Docker Containers Have Been Stopped Successfully ==="

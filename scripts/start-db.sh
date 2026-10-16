#!/bin/bash
set -e

echo "=== Microservices Colima Helper (PostgreSQL + Redis) ==="

# Check if colima is running
if ! colima status >/dev/null 2>&1; then
    echo "[Colima] Starting Colima runtime..."
    colima start
else
    echo "[Colima] Colima runtime is active."
fi

PG_CONTAINER="microservices-postgres"
REDIS_CONTAINER="microservices-redis"

# 1. PostgreSQL Container (Port 5433)
if docker ps -a | grep -w "${PG_CONTAINER}" >/dev/null 2>&1; then
    if docker ps | grep -w "${PG_CONTAINER}" >/dev/null 2>&1; then
        echo "[Docker] PostgreSQL container '${PG_CONTAINER}' is already running on port 5433."
    else
        echo "[Docker] Starting existing PostgreSQL container '${PG_CONTAINER}'..."
        docker start ${PG_CONTAINER}
    fi
else
    echo "[Docker] Creating and launching new PostgreSQL container '${PG_CONTAINER}' on port 5433..."
    docker run -d \
      --name ${PG_CONTAINER} \
      -e POSTGRES_USER=postgres \
      -e POSTGRES_PASSWORD=postgres \
      -e POSTGRES_DB=microservices_db \
      -p 5433:5432 \
      postgres:15-alpine
fi

# 2. Redis Container (Port 6379)
if docker ps -a | grep -w "${REDIS_CONTAINER}" >/dev/null 2>&1; then
    if docker ps | grep -w "${REDIS_CONTAINER}" >/dev/null 2>&1; then
        echo "[Docker] Redis container '${REDIS_CONTAINER}' is already running on port 6379."
    else
        echo "[Docker] Starting existing Redis container '${REDIS_CONTAINER}'..."
        docker start ${REDIS_CONTAINER}
    fi
else
    echo "[Docker] Creating and launching new Redis container '${REDIS_CONTAINER}' on port 6379..."
    docker run -d \
      --name ${REDIS_CONTAINER} \
      -p 6379:6379 \
      redis:7-alpine
fi

echo "=== Infrastructure Services Ready: PostgreSQL (localhost:5433), Redis (localhost:6379) ==="

# Go Multi-Microservices Platform v0.0.1

Production-ready, highly decoupled Go microservices platform featuring **Ed25519 (EdDSA) Dual-Token JWT Authentication**, **Real-Time Redis Access Token Blacklisting**, **gRPC Inter-Service Communication**, and a dedicated **PostgreSQL Database Microservice** (`db-service`).

---

## 🏛️ System Architecture Overview

```
                      +-------------------+
                      |    HTTP Client    |
                      +---------+---------+
                                |
          +---------------------+---------------------+
          | HTTP                                      | HTTP
          v                                           v
+------------------+                        +-------------------+
|   Auth Service   |                        | Protected Service |
|   (Port :8081)   |                        |   (Port :8082)    |
|                  |                        |                   |
| - POST /register |                        | - GET /protected  |
| - POST /login    |                        |                   |
| - POST /renew    |                        | [Auth Middleware] |
| - POST /logout   |                        | - Verifies Ed25519|
| - GET /jwks.json |                        |   statelessly     |
|                  |                        | - Checks Redis    |
| (Ed25519 Private |                        |   Blacklist       |
|  Key Signing)    |                        +---------+---------+
+--------+---------+                                  |
         |                                            |
         | gRPC Server (:50051)                       |
         +<-------------------------------------------+
         | 1) GetJWKS()
         | 2) GetUserProfile()
         |
         | gRPC Client
         v
+------------------+                        +-------------------+
|    DB Service    |  <--- gRPC (:50052)    |  Redis Cache DB   |
|  (PostgreSQL DB) |                        |   (Port :6379)    |
|                  |                        |                   |
| - Owns DB CRUD & |                        | - Stores JTI      |
|   Refresh Tokens |                        |   Blacklists      |
+--------+---------+                        +-------------------+
         |
         v
+------------------+
|  PostgreSQL DB   |  (Port :5433, managed via Colima CLI)
+------------------+
```

---

## 🚀 Management Commands

### 1. Start Infrastructure & Microservices

Start PostgreSQL & Redis containers via Colima:
```bash
./scripts/start-db.sh
```

Launch Microservices in separate terminal tabs:
```bash
# Terminal 1: DB Service
cd db-service && go run cmd/main.go

# Terminal 2: Auth Service
cd auth-service && go run cmd/main.go

# Terminal 3: Protected Service
cd protected-service && go run cmd/main.go
```

### 2. Stop All Microservices & Docker Containers

To cleanly stop all background Go microservices processes and container instances:

```bash
./scripts/stop-all.sh
```

---

## 📄 Postman Collection

Import [`microservices_v0.0.1.postman_collection.json`](file:///Users/umar-indriver/Desktop/go-lang/go-microservices/microservices_v0.0.1.postman_collection.json) into Postman / Insomnia for automated testing of registration, login, token renewal, Ed25519 JWKS verification, and Redis logout blacklisting.
# go-microservices-exmaple

# Multi-stage Dockerfile for auth-service
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy common module first
COPY common ../common
COPY auth-service/go.mod auth-service/go.sum* ./
RUN go mod download

COPY auth-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /auth-service .
COPY auth-service/config ./config

EXPOSE 8081 50051
CMD ["./auth-service"]

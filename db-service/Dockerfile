# Multi-stage Dockerfile for db-service
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy common module first
COPY common ../common
COPY db-service/go.mod db-service/go.sum* ./
RUN go mod download

COPY db-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /db-service ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /db-service .
COPY db-service/config ./config

EXPOSE 50052
CMD ["./db-service"]

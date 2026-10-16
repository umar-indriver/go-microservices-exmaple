module github.com/microservices/auth-service

go 1.25.0

replace github.com/microservices/common => ../common

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/microservices/common v0.0.0-000000000000-000000000000
	github.com/redis/go-redis/v9 v9.5.1
	golang.org/x/crypto v0.54.0
	google.golang.org/grpc v1.84.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr string) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password by default
		DB:       0,  // default DB
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis server at %s: %w", addr, err)
	}

	log.Printf("[Redis] Successfully connected to Redis at %s", addr)
	return &RedisClient{client: rdb}, nil
}

func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *RedisClient) BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" {
		return nil
	}
	key := fmt.Sprintf("blacklist:jti:%s", jti)
	return r.client.Set(ctx, key, "revoked", ttl).Err()
}

func (r *RedisClient) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	key := fmt.Sprintf("blacklist:jti:%s", jti)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "revoked", nil
}

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/microservices/protected-service/internal/jwks"
	"github.com/microservices/protected-service/internal/redis"
)

type contextKey string

const UserContextKey contextKey = "user_context"

type UserContext struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

type CustomClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type AuthMiddleware struct {
	cache       *jwks.JWKSCache
	redisClient *redis.RedisClient
}

func NewAuthMiddleware(cache *jwks.JWKSCache, redisClient *redis.RedisClient) *AuthMiddleware {
	return &AuthMiddleware{
		cache:       cache,
		redisClient: redisClient,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error":"invalid Authorization header format, expected 'Bearer <token>'"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]

		token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok && token.Method.Alg() != "EdDSA" {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			kid, ok := token.Header["kid"].(string)
			if !ok || kid == "" {
				return nil, fmt.Errorf("missing key id 'kid' in token header")
			}

			pubKey, err := m.cache.GetPublicKey(kid)
			if err != nil {
				return nil, err
			}

			return pubKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized: %v"}`, err), http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*CustomClaims)
		if !ok {
			http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
			return
		}

		// Check Redis Blacklist for revoked JTI
		jti := claims.ID
		if m.redisClient != nil && jti != "" {
			blacklisted, err := m.redisClient.IsBlacklisted(r.Context(), jti)
			if err == nil && blacklisted {
				http.Error(w, `{"error":"unauthorized: token has been blacklisted / logged out"}`, http.StatusUnauthorized)
				return
			}
		}

		userCtx := UserContext{
			UserID: claims.Subject,
			Email:  claims.Email,
			Name:   claims.Name,
			Role:   claims.Role,
		}

		ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserFromContext(ctx context.Context) (UserContext, bool) {
	u, ok := ctx.Value(UserContextKey).(UserContext)
	return u, ok
}

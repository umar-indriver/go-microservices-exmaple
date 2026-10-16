package httphandler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
	"github.com/microservices/auth-service/internal/client"
	authjwt "github.com/microservices/auth-service/internal/jwt"
	"github.com/microservices/auth-service/internal/keys"
	"github.com/microservices/auth-service/internal/redis"
	"golang.org/x/crypto/bcrypt"
)

type HTTPHandler struct {
	dbClient     *client.DBClient
	tokenService *authjwt.TokenService
	keyManager   *keys.KeyManager
	redisClient  *redis.RedisClient
}

func NewHTTPHandler(dbClient *client.DBClient, tokenService *authjwt.TokenService, keyManager *keys.KeyManager, redisClient *redis.RedisClient) *HTTPHandler {
	return &HTTPHandler{
		dbClient:     dbClient,
		tokenService: tokenService,
		keyManager:   keyManager,
		redisClient:  redisClient,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RenewRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

type AuthResponse struct {
	Tokens *authjwt.TokenPair `json:"tokens"`
	User   UserPayload        `json:"user"`
}

type UserPayload struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.HandleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", h.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/renew", h.HandleRenew)
	mux.HandleFunc("POST /api/v1/auth/logout", h.HandleLogout)
	mux.HandleFunc("GET /.well-known/jwks.json", h.HandleJWKS)
}

func (h *HTTPHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		http.Error(w, `{"error":"email, password, and name are required"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.dbClient.CreateUser(r.Context(), req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		http.Error(w, fmtJSONError(err.Error()), http.StatusInternalServerError)
		return
	}

	pair, refreshHash, expAt, err := h.tokenService.GenerateTokenPair(resp.GetId(), resp.GetEmail(), resp.GetName(), resp.GetRole())
	if err != nil {
		http.Error(w, `{"error":"failed to generate token pair"}`, http.StatusInternalServerError)
		return
	}

	_, err = h.dbClient.SaveRefreshToken(r.Context(), resp.GetId(), refreshHash, expAt)
	if err != nil {
		http.Error(w, `{"error":"failed to persist refresh token"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{
		Tokens: pair,
		User: UserPayload{
			ID:    resp.GetId(),
			Email: resp.GetEmail(),
			Name:  resp.GetName(),
			Role:  resp.GetRole(),
		},
	})
}

func (h *HTTPHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password are required"}`, http.StatusBadRequest)
		return
	}

	userResp, err := h.dbClient.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userResp.GetPasswordHash()), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	pair, refreshHash, expAt, err := h.tokenService.GenerateTokenPair(userResp.GetId(), userResp.GetEmail(), userResp.GetName(), userResp.GetRole())
	if err != nil {
		http.Error(w, `{"error":"failed to generate token pair"}`, http.StatusInternalServerError)
		return
	}

	_, err = h.dbClient.SaveRefreshToken(r.Context(), userResp.GetId(), refreshHash, expAt)
	if err != nil {
		http.Error(w, `{"error":"failed to persist refresh token"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		Tokens: pair,
		User: UserPayload{
			ID:    userResp.GetId(),
			Email: userResp.GetEmail(),
			Name:  userResp.GetName(),
			Role:  userResp.GetRole(),
		},
	})
}

func (h *HTTPHandler) HandleRenew(w http.ResponseWriter, r *http.Request) {
	var req RenewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		http.Error(w, `{"error":"refresh_token is required"}`, http.StatusBadRequest)
		return
	}

	tokenHash := authjwt.HashToken(req.RefreshToken)
	rtResp, err := h.dbClient.GetRefreshToken(r.Context(), tokenHash)
	if err != nil || rtResp.GetRevoked() {
		http.Error(w, `{"error":"invalid or revoked refresh token"}`, http.StatusUnauthorized)
		return
	}

	expTime, err := time.Parse(time.RFC3339, rtResp.GetExpiresAt())
	if err != nil || time.Now().After(expTime) {
		http.Error(w, `{"error":"refresh token expired"}`, http.StatusUnauthorized)
		return
	}

	// Revoke old refresh token
	_ = h.dbClient.RevokeRefreshToken(r.Context(), tokenHash, "")

	// Fetch user info
	userResp, err := h.dbClient.GetUserByID(r.Context(), rtResp.GetUserId())
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusUnauthorized)
		return
	}

	// Issue new Access + Refresh Token pair
	pair, newRefreshHash, newExpAt, err := h.tokenService.GenerateTokenPair(userResp.GetId(), userResp.GetEmail(), userResp.GetName(), userResp.GetRole())
	if err != nil {
		http.Error(w, `{"error":"failed to generate token pair"}`, http.StatusInternalServerError)
		return
	}

	_, _ = h.dbClient.SaveRefreshToken(r.Context(), userResp.GetId(), newRefreshHash, newExpAt)

	writeJSON(w, http.StatusOK, AuthResponse{
		Tokens: pair,
		User: UserPayload{
			ID:    userResp.GetId(),
			Email: userResp.GetEmail(),
			Name:  userResp.GetName(),
			Role:  userResp.GetRole(),
		},
	})
}

func (h *HTTPHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	// 1. Revoke Refresh Token in PostgreSQL if provided
	if req.RefreshToken != "" {
		tokenHash := authjwt.HashToken(req.RefreshToken)
		_ = h.dbClient.RevokeRefreshToken(r.Context(), tokenHash, "")
	}

	// 2. Blacklist Access Token JTI in Redis if Authorization header is provided
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			tokenStr := parts[1]
			parser := golangjwt.NewParser()
			token, _, err := parser.ParseUnverified(tokenStr, &golangjwt.RegisteredClaims{})
			if err == nil && token != nil {
				if claims, ok := token.Claims.(*golangjwt.RegisteredClaims); ok {
					jti := claims.ID
					var ttl time.Duration = 15 * time.Minute
					if claims.ExpiresAt != nil {
						rem := time.Until(claims.ExpiresAt.Time)
						if rem > 0 {
							ttl = rem
						}
					}
					if h.redisClient != nil && jti != "" {
						_ = h.redisClient.BlacklistJTI(r.Context(), jti, ttl)
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "successfully logged out and blacklisted token in Redis"})
}

func (h *HTTPHandler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.keyManager.GetJWKSJSON()))
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func fmtJSONError(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

package jwt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/microservices/auth-service/internal/keys"
)

type CustomClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	TokenType             string    `json:"token_type"`
}

type TokenService struct {
	keyManager       *keys.KeyManager
	issuer           string
	accessExpMinutes int
	refreshExpDays   int
}

func NewTokenService(km *keys.KeyManager, issuer string, accessExpMinutes, refreshExpDays int) *TokenService {
	return &TokenService{
		keyManager:       km,
		issuer:           issuer,
		accessExpMinutes: accessExpMinutes,
		refreshExpDays:   refreshExpDays,
	}
}

func (s *TokenService) GenerateTokenPair(userID, email, name, role string) (*TokenPair, string, time.Time, error) {
	now := time.Now().UTC()
	accessExp := now.Add(time.Duration(s.accessExpMinutes) * time.Minute)
	refreshExp := now.Add(time.Duration(s.refreshExpDays*24) * time.Hour)

	jti := uuid.New().String()

	claims := CustomClaims{
		Email: email,
		Name:  name,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = s.keyManager.KeyID

	accessTokenStr, err := token.SignedString(s.keyManager.PrivateKey)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("failed to sign Ed25519 access token: %w", err)
	}

	rawRefreshToken := uuid.New().String() + "." + uuid.New().String()
	refreshTokenHash := HashToken(rawRefreshToken)

	pair := &TokenPair{
		AccessToken:           accessTokenStr,
		RefreshToken:          rawRefreshToken,
		AccessTokenExpiresAt:  accessExp,
		RefreshTokenExpiresAt: refreshExp,
		TokenType:             "Bearer",
	}

	return pair, refreshTokenHash, refreshExp, nil
}

func HashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

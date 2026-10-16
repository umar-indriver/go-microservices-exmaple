package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
)

type KeyManager struct {
	mu         sync.RWMutex
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	KeyID      string
	jwksJSON   string
}

type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	X   string `json:"x"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

func NewKeyManager() (*KeyManager, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}

	keyID := "auth-eddsa-key-1"
	km := &KeyManager{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		KeyID:      keyID,
	}

	if err := km.buildJWKS(); err != nil {
		return nil, err
	}

	return km, nil
}

func (km *KeyManager) buildJWKS() error {
	xStr := base64.RawURLEncoding.EncodeToString(km.PublicKey)

	jwk := JWK{
		Kty: "OKP",
		Crv: "Ed25519",
		Alg: "EdDSA",
		Kid: km.KeyID,
		Use: "sig",
		X:   xStr,
	}

	jwks := JWKS{
		Keys: []JWK{jwk},
	}

	data, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	km.mu.Lock()
	km.jwksJSON = string(data)
	km.mu.Unlock()

	return nil
}

func (km *KeyManager) GetJWKSJSON() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.jwksJSON
}

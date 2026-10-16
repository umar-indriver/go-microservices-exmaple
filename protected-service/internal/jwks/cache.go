package jwks

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	ErrKeyNotFound = errors.New("public key for kid not found in JWKS cache")
)

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

type JWKSCache struct {
	mu          sync.RWMutex
	authHTTPURL string
	keys        map[string]ed25519.PublicKey
	lastFetched time.Time
	ttl         time.Duration
	client      *http.Client
}

func NewJWKSCache(authHTTPURL string, ttl time.Duration) *JWKSCache {
	return &JWKSCache{
		authHTTPURL: authHTTPURL,
		keys:        make(map[string]ed25519.PublicKey),
		ttl:         ttl,
		client:      &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *JWKSCache) GetPublicKey(kid string) (ed25519.PublicKey, error) {
	c.mu.RLock()
	key, exists := c.keys[kid]
	expired := time.Since(c.lastFetched) > c.ttl
	c.mu.RUnlock()

	if exists && !expired {
		return key, nil
	}

	// Fetch or refresh JWKS from Auth service
	if err := c.RefreshKeys(); err != nil {
		if exists {
			return key, nil
		}
		return nil, fmt.Errorf("failed to fetch JWKS and no cached key found: %w", err)
	}

	c.mu.RLock()
	key, exists = c.keys[kid]
	c.mu.RUnlock()

	if !exists {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

func (c *JWKSCache) RefreshKeys() error {
	url := fmt.Sprintf("%s/.well-known/jwks.json", c.authHTTPURL)
	resp, err := c.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed HTTP GET for JWKS at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d from JWKS endpoint", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS JSON: %w", err)
	}

	newKeys := make(map[string]ed25519.PublicKey)
	for _, jwk := range jwks.Keys {
		pubKey, err := parseEd25519PublicKey(jwk)
		if err != nil {
			continue
		}
		newKeys[jwk.Kid] = pubKey
	}

	c.mu.Lock()
	c.keys = newKeys
	c.lastFetched = time.Now()
	c.mu.Unlock()

	return nil
}

func parseEd25519PublicKey(jwk JWK) (ed25519.PublicKey, error) {
	if jwk.Kty != "OKP" || jwk.Crv != "Ed25519" {
		return nil, fmt.Errorf("unsupported key type or curve: kty=%s, crv=%s", jwk.Kty, jwk.Crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key X: %w", err)
	}

	if len(xBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size: got %d bytes, expected %d", len(xBytes), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(xBytes), nil
}

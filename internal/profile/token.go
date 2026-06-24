package profile

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytetrade/hass-cli/internal/keychain"
)

// TokenStore persists Home Assistant long-lived access tokens in the OS
// keychain, keyed by profile name (the keychain account).
type TokenStore struct {
	access keychain.KeychainAccess
}

// NewTokenStore returns a TokenStore backed by the real OS keychain.
func NewTokenStore() *TokenStore {
	return &TokenStore{access: keychain.Default()}
}

// NewTokenStoreWith lets tests inject an in-memory keychain.
func NewTokenStoreWith(access keychain.KeychainAccess) *TokenStore {
	return &TokenStore{access: access}
}

// Get returns the token for a profile. Returns ErrTokenNotFound when absent.
func (s *TokenStore) Get(name string) (string, error) {
	v, err := s.access.Get(keychain.HassCliService, name)
	if err != nil {
		return "", err
	}
	if v == "" {
		return "", ErrTokenNotFound
	}
	return v, nil
}

// Set stores (overwrites) the token for a profile.
func (s *TokenStore) Set(name, token string) error {
	return s.access.Set(keychain.HassCliService, name, token)
}

// Delete removes the token for a profile. Deleting a missing entry is a no-op.
func (s *TokenStore) Delete(name string) error {
	return s.access.Remove(keychain.HassCliService, name)
}

// ErrTokenNotFound is returned when no token is stored for a profile.
var ErrTokenNotFound = errors.New("token not found")

// ErrNoExpClaim is returned by ExpiresAt when the JWT payload has no exp field.
var ErrNoExpClaim = errors.New("jwt has no exp claim")

// ExpiresAt decodes only the `exp` claim of a JWT and returns it as a
// time.Time. It does NOT verify the signature; use the result as a
// client-side hint only. HA long-lived tokens are JWTs with a far-future exp.
func ExpiresAt(token string) (time.Time, error) {
	if token == "" {
		return time.Time{}, errors.New("token is empty")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("token does not look like a JWT (want 3 segments, got %d)", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("decode payload: %w", err)
		}
	}
	var c struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &c); err != nil {
		return time.Time{}, fmt.Errorf("parse payload: %w", err)
	}
	if c.Exp == 0 {
		return time.Time{}, ErrNoExpClaim
	}
	return time.Unix(c.Exp, 0), nil
}

// Status describes what the local token store can prove about a profile
// without a network call.
func Status(store *TokenStore, name string, now time.Time) string {
	tok, err := store.Get(name)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return "never"
		}
		return "unknown"
	}
	exp, err := ExpiresAt(tok)
	if err != nil {
		if errors.Is(err, ErrNoExpClaim) {
			return "logged-in"
		}
		return "logged-in (unparseable token)"
	}
	if !now.Before(exp) {
		return "expired"
	}
	return "logged-in"
}

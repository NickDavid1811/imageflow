// Package application implements API-key authentication use cases.
package application

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var ErrInvalidAPIKey = errors.New("invalid API key")

// APIKey is the stored identity material used during authentication.
type APIKey struct{ ID, ProjectID, Environment, SecretHash string }

// APIKeyRepository retrieves and records API-key usage.
type APIKeyRepository interface {
	FindActiveByPrefix(context.Context, string) (APIKey, error)
	TouchLastUsed(context.Context, string, time.Time) error
}

// Principal is the authenticated project identity attached to a request.
type Principal struct{ ProjectID, Environment, APIKeyID string }

// Authenticator verifies opaque bearer tokens without storing their plaintext.
type Authenticator struct {
	repository APIKeyRepository
	now        func() time.Time
}

// NewAuthenticator constructs an API-key authenticator.
func NewAuthenticator(repository APIKeyRepository) *Authenticator {
	return &Authenticator{repository: repository, now: time.Now}
}

// Authenticate verifies one ImageFlow API key and returns its project identity.
func (authenticator *Authenticator) Authenticate(ctx context.Context, token string) (Principal, error) {
	token = strings.TrimSpace(token)
	if len(token) < 24 || (!strings.HasPrefix(token, "if_test_") && !strings.HasPrefix(token, "if_live_")) {
		return Principal{}, ErrInvalidAPIKey
	}
	prefix := token
	if len(prefix) > 24 {
		prefix = prefix[:24]
	}
	key, err := authenticator.repository.FindActiveByPrefix(ctx, prefix)
	if err != nil {
		return Principal{}, ErrInvalidAPIKey
	}
	sum := sha256.Sum256([]byte(token))
	provided := hex.EncodeToString(sum[:])
	if len(provided) != len(key.SecretHash) || subtle.ConstantTimeCompare([]byte(provided), []byte(key.SecretHash)) != 1 {
		return Principal{}, ErrInvalidAPIKey
	}
	if err := authenticator.repository.TouchLastUsed(ctx, key.ID, authenticator.now().UTC()); err != nil {
		return Principal{}, err
	}
	return Principal{ProjectID: key.ProjectID, Environment: key.Environment, APIKeyID: key.ID}, nil
}

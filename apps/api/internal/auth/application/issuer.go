package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Project is an isolated ImageFlow tenant.
type Project struct {
	ID, Name  string
	CreatedAt time.Time
}

// IssuedAPIKey contains persistable metadata and the one-time plaintext token.
type IssuedAPIKey struct {
	APIKey
	Name, LookupPrefix, Token string
	CreatedAt                 time.Time
}

// CredentialWriter atomically persists a project and its first API key.
type CredentialWriter interface {
	CreateProjectWithAPIKey(context.Context, Project, IssuedAPIKey) error
}

// Issuer provisions projects and high-entropy API keys.
type Issuer struct {
	repository CredentialWriter
	now        func() time.Time
}

// NewIssuer creates a credential issuer.
func NewIssuer(repository CredentialWriter) *Issuer {
	return &Issuer{repository: repository, now: time.Now}
}

// CreateProject provisions one project and its initial test or live key.
func (issuer *Issuer) CreateProject(ctx context.Context, name, environment string) (Project, IssuedAPIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, IssuedAPIKey{}, errors.New("project name is required")
	}
	if environment != "test" && environment != "live" {
		return Project{}, IssuedAPIKey{}, errors.New("environment must be test or live")
	}
	projectRandom, err := randomHex(8)
	if err != nil {
		return Project{}, IssuedAPIKey{}, err
	}
	secret, err := randomHex(24)
	if err != nil {
		return Project{}, IssuedAPIKey{}, err
	}
	keyRandom, err := randomHex(8)
	if err != nil {
		return Project{}, IssuedAPIKey{}, err
	}
	token := "if_" + environment + "_" + secret
	prefix := token
	if len(prefix) > 24 {
		prefix = prefix[:24]
	}
	sum := sha256.Sum256([]byte(token))
	now := issuer.now().UTC()
	project := Project{ID: "prj_" + projectRandom, Name: name, CreatedAt: now}
	key := IssuedAPIKey{APIKey: APIKey{ID: "key_" + keyRandom, ProjectID: project.ID, Environment: environment, SecretHash: hex.EncodeToString(sum[:])}, Name: "Initial " + environment + " key", LookupPrefix: prefix, Token: token, CreatedAt: now}
	if err := issuer.repository.CreateProjectWithAPIKey(ctx, project, key); err != nil {
		return Project{}, IssuedAPIKey{}, fmt.Errorf("store credentials: %w", err)
	}
	return project, key, nil
}

func randomHex(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

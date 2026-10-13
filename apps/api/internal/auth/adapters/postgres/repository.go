// Package postgres persists API-key identities in PostgreSQL.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	authapplication "github.com/nickdavid1811/imageflow/internal/auth/application"
)

// EnsureDevelopmentAPIKey idempotently installs the explicitly configured local credential.
func (repository *Repository) EnsureDevelopmentAPIKey(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	prefix := token
	if len(prefix) > 24 {
		prefix = prefix[:24]
	}
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	_, err := repository.pool.Exec(ctx, `INSERT INTO api_keys(id,project_id,name,environment,lookup_prefix,secret_hash,created_at) VALUES('key_local','prj_local','Local console','test',$1,$2,now()) ON CONFLICT (id) DO UPDATE SET lookup_prefix=EXCLUDED.lookup_prefix, secret_hash=EXCLUDED.secret_hash, revoked_at=NULL`, prefix, hash)
	if err != nil {
		return fmt.Errorf("ensure development API key: %w", err)
	}
	return nil
}

// Repository is the PostgreSQL API-key adapter.
type Repository struct{ pool *pgxpool.Pool }

// NewRepository creates an API-key repository.
func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// FindActiveByPrefix returns a non-revoked key candidate.
func (repository *Repository) FindActiveByPrefix(ctx context.Context, prefix string) (authapplication.APIKey, error) {
	var key authapplication.APIKey
	err := repository.pool.QueryRow(ctx, `SELECT id, project_id, environment, secret_hash FROM api_keys WHERE lookup_prefix=$1 AND revoked_at IS NULL`, prefix).Scan(&key.ID, &key.ProjectID, &key.Environment, &key.SecretHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return authapplication.APIKey{}, authapplication.ErrInvalidAPIKey
	}
	if err != nil {
		return authapplication.APIKey{}, fmt.Errorf("find API key: %w", err)
	}
	return key, nil
}

// TouchLastUsed records usage at most once every five minutes.
func (repository *Repository) TouchLastUsed(ctx context.Context, id string, now time.Time) error {
	_, err := repository.pool.Exec(ctx, `UPDATE api_keys SET last_used_at=$2 WHERE id=$1 AND (last_used_at IS NULL OR last_used_at < $2::timestamptz - interval '5 minutes')`, id, now)
	if err != nil {
		return fmt.Errorf("touch API key: %w", err)
	}
	return nil
}

// CreateProjectWithAPIKey atomically persists a project and its initial credential.
func (repository *Repository) CreateProjectWithAPIKey(ctx context.Context, project authapplication.Project, key authapplication.IssuedAPIKey) error {
	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err = transaction.Exec(ctx, `INSERT INTO projects(id,name,created_at) VALUES($1,$2,$3)`, project.ID, project.Name, project.CreatedAt); err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	if _, err = transaction.Exec(ctx, `INSERT INTO api_keys(id,project_id,name,environment,lookup_prefix,secret_hash,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, key.ID, key.ProjectID, key.Name, key.Environment, key.LookupPrefix, key.SecretHash, key.CreatedAt); err != nil {
		return fmt.Errorf("insert API key: %w", err)
	}
	return transaction.Commit(ctx)
}

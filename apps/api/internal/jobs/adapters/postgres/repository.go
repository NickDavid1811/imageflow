package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickdavid1811/imageflow/internal/jobs/application"
	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (repository *JobRepository) Create(ctx context.Context, job domain.Job) error {
	operations, err := json.Marshal(job.Operations)
	if err != nil {
		return fmt.Errorf("encode operations: %w", err)
	}
	eventPayload, err := json.Marshal(map[string]string{"job_id": job.ID})
	if err != nil {
		return fmt.Errorf("encode outbox event: %w", err)
	}

	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	_, err = transaction.Exec(ctx, `
		INSERT INTO jobs (
			id, source_url, operations, status, progress, attempts,
			error_code, error_message, created_at, started_at, completed_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9, $10, $11, $12)`,
		job.ID, job.SourceURL, operations, job.Status, job.Progress, job.Attempts,
		job.ErrorCode, job.ErrorMessage, job.CreatedAt, job.StartedAt, job.CompletedAt, job.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	_, err = transaction.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_id, event_type, payload, created_at)
		VALUES ($1, 'job.requested.v1', $2, $3)`, job.ID, eventPayload, job.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (repository *JobRepository) FindByID(ctx context.Context, id string) (domain.Job, error) {
	var job domain.Job
	var operations []byte
	err := repository.pool.QueryRow(ctx, `
		SELECT id, source_url, operations, status, progress, attempts,
		       COALESCE(error_code, ''), COALESCE(error_message, ''),
		       created_at, started_at, completed_at, updated_at
		FROM jobs
		WHERE id = $1`, id).Scan(
		&job.ID, &job.SourceURL, &operations, &job.Status, &job.Progress, &job.Attempts,
		&job.ErrorCode, &job.ErrorMessage, &job.CreatedAt, &job.StartedAt, &job.CompletedAt, &job.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, application.ErrJobNotFound
	}
	if err != nil {
		return domain.Job{}, fmt.Errorf("query job: %w", err)
	}
	if err := json.Unmarshal(operations, &job.Operations); err != nil {
		return domain.Job{}, fmt.Errorf("decode operations: %w", err)
	}
	return job, nil
}

package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nickdavid1811/imageflow/internal/outbox/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) ClaimBatch(ctx context.Context, limit int, now time.Time, lease time.Duration) ([]domain.Event, error) {
	rows, err := repository.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id
			FROM outbox_events
			WHERE published_at IS NULL
			  AND available_at <= $1
			  AND (locked_at IS NULL OR locked_at < $1 - make_interval(secs => $3))
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE outbox_events AS events
		SET locked_at = $1, attempts = attempts + 1
		FROM candidates
		WHERE events.id = candidates.id
		RETURNING events.id, events.aggregate_id, events.event_type,
		          events.payload, events.attempts, events.created_at`,
		now.UTC(), limit, int(lease.Seconds()),
	)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		var event domain.Event
		if err := rows.Scan(&event.ID, &event.AggregateID, &event.Type, &event.Payload, &event.Attempts, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}
	return events, nil
}

func (repository *Repository) MarkPublished(ctx context.Context, id int64, now time.Time) error {
	_, err := repository.pool.Exec(ctx, `
		UPDATE outbox_events
		SET published_at = $2, locked_at = NULL, last_error = NULL
		WHERE id = $1 AND published_at IS NULL`, id, now.UTC())
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	return nil
}

func (repository *Repository) MarkFailed(ctx context.Context, id int64, message string, availableAt time.Time) error {
	_, err := repository.pool.Exec(ctx, `
		UPDATE outbox_events
		SET locked_at = NULL, last_error = $2, available_at = $3
		WHERE id = $1 AND published_at IS NULL`, id, message, availableAt.UTC())
	if err != nil {
		return fmt.Errorf("release outbox event: %w", err)
	}
	return nil
}

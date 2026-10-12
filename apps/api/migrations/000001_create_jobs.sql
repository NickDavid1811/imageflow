-- +imageflow Up
CREATE TABLE jobs (
    id              TEXT PRIMARY KEY,
    source_url      TEXT NOT NULL,
    operations      JSONB NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'cancelled')),
    progress        SMALLINT NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
    attempts        INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    error_code      TEXT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX jobs_status_created_at_idx ON jobs (status, created_at);

CREATE TABLE outbox_events (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_id    TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    published_at    TIMESTAMPTZ
);

CREATE INDEX outbox_events_unpublished_idx
    ON outbox_events (id)
    WHERE published_at IS NULL;

-- +imageflow Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS jobs;

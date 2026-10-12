-- +imageflow Up
ALTER TABLE outbox_events
    ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    ADD COLUMN available_at TIMESTAMPTZ,
    ADD COLUMN locked_at TIMESTAMPTZ,
    ADD COLUMN last_error TEXT;

UPDATE outbox_events SET available_at = created_at WHERE available_at IS NULL;
ALTER TABLE outbox_events ALTER COLUMN available_at SET NOT NULL;

DROP INDEX outbox_events_unpublished_idx;
CREATE INDEX outbox_events_delivery_idx
    ON outbox_events (available_at, id)
    WHERE published_at IS NULL;

-- +imageflow Down
DROP INDEX IF EXISTS outbox_events_delivery_idx;
CREATE INDEX outbox_events_unpublished_idx
    ON outbox_events (id)
    WHERE published_at IS NULL;

ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS locked_at,
    DROP COLUMN IF EXISTS available_at,
    DROP COLUMN IF EXISTS attempts;

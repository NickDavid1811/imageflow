-- +imageflow Up
ALTER TABLE outbox_events
    ALTER COLUMN available_at SET DEFAULT now();

-- +imageflow Down
ALTER TABLE outbox_events
    ALTER COLUMN available_at DROP DEFAULT;

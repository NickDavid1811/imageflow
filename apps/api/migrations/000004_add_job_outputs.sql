-- +imageflow Up
ALTER TABLE jobs
    ADD COLUMN output_key TEXT,
    ADD COLUMN output_content_type TEXT,
    ADD COLUMN output_format TEXT,
    ADD COLUMN output_size_bytes BIGINT,
    ADD COLUMN output_width INTEGER,
    ADD COLUMN output_height INTEGER;

-- +imageflow Down
ALTER TABLE jobs
    DROP COLUMN output_height,
    DROP COLUMN output_width,
    DROP COLUMN output_size_bytes,
    DROP COLUMN output_format,
    DROP COLUMN output_content_type,
    DROP COLUMN output_key;

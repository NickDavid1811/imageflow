-- +imageflow Up
CREATE TABLE projects (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
);

CREATE TABLE api_keys (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    environment     TEXT NOT NULL CHECK (environment IN ('test', 'live')),
    lookup_prefix   TEXT NOT NULL UNIQUE,
    secret_hash     CHAR(64) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    last_used_at    TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ
);

INSERT INTO projects (id, name, created_at)
VALUES ('prj_local', 'Local development', now());

ALTER TABLE jobs ADD COLUMN project_id TEXT;
UPDATE jobs SET project_id = 'prj_local' WHERE project_id IS NULL;
ALTER TABLE jobs ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE jobs ADD CONSTRAINT jobs_project_id_fkey FOREIGN KEY (project_id) REFERENCES projects(id);
CREATE INDEX jobs_project_created_at_idx ON jobs (project_id, created_at DESC);

-- +imageflow Down
DROP INDEX IF EXISTS jobs_project_created_at_idx;
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_project_id_fkey;
ALTER TABLE jobs DROP COLUMN IF EXISTS project_id;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS projects;

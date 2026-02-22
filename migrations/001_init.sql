CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    trigger     JSONB NOT NULL,
    action      JSONB NOT NULL,
    status      TEXT NOT NULL DEFAULT 'scheduled'
                    CHECK (status IN ('scheduled', 'cancelled', 'completed')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_run    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tasks_status   ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_next_run ON tasks (next_run) WHERE status = 'scheduled';

CREATE TABLE IF NOT EXISTS task_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id          UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    run_at           TIMESTAMPTZ NOT NULL,
    status_code      INTEGER NOT NULL DEFAULT 0,
    success          BOOLEAN NOT NULL DEFAULT FALSE,
    response_headers JSONB,
    response_body    TEXT,
    error_message    TEXT,
    duration_ms      BIGINT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_results_task_id ON task_results (task_id);
CREATE INDEX IF NOT EXISTS idx_task_results_run_at  ON task_results (run_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_results_success ON task_results (success);
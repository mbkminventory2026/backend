-- Initial foundation table to validate migration + sqlc workflow.
CREATE TABLE IF NOT EXISTS health_checks (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

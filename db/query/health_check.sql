-- name: CreateHealthCheck :one
INSERT INTO health_checks DEFAULT VALUES
RETURNING id, created_at;

-- name: GetLatestHealthCheck :one
SELECT id, created_at
FROM health_checks
ORDER BY id DESC
LIMIT 1;

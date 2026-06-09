-- name: GetProductionStatusPlanByID :one
SELECT * FROM PRODUCTION_STATUS_PLAN WHERE id_production_status_plan = $1 LIMIT 1;

-- name: ListProductionStatusPlans :many
SELECT * FROM PRODUCTION_STATUS_PLAN
WHERE (
    sqlc.arg(search_term)::text = '' OR
    name ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'name' AND NOT sqlc.arg(sort_desc)::bool THEN name END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_desc)::bool THEN name END DESC,
    created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProductionStatusPlans :one
SELECT COUNT(*) FROM PRODUCTION_STATUS_PLAN
WHERE (
    sqlc.arg(search_term)::text = '' OR
    name ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: CreateProductionStatusPlan :one
INSERT INTO PRODUCTION_STATUS_PLAN (name)
VALUES ($1) RETURNING *;

-- name: UpdateProductionStatusPlan :one
UPDATE PRODUCTION_STATUS_PLAN
SET name = $2
WHERE id_production_status_plan = $1
RETURNING *;

-- name: DeleteProductionStatusPlan :execrows
DELETE FROM PRODUCTION_STATUS_PLAN
WHERE id_production_status_plan = $1;

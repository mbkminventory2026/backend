-- name: GetProductionLineByID :one
SELECT * FROM PRODUCTION_LINE WHERE id_production_line = $1 LIMIT 1;

-- name: ListProductionLines :many
SELECT * FROM PRODUCTION_LINE
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

-- name: CountProductionLines :one
SELECT COUNT(*) FROM PRODUCTION_LINE
WHERE (
    sqlc.arg(search_term)::text = '' OR
    name ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: CreateProductionLine :one
INSERT INTO PRODUCTION_LINE (name)
VALUES ($1) RETURNING *;

-- name: UpdateProductionLine :one
UPDATE PRODUCTION_LINE
SET name = $2
WHERE id_production_line = $1
RETURNING *;

-- name: DeleteProductionLine :execrows
DELETE FROM PRODUCTION_LINE
WHERE id_production_line = $1;

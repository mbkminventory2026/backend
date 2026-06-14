-- name: GetSizeByID :one
SELECT *
FROM MASTER_SIZE
WHERE id_size = $1
LIMIT 1;

-- name: GetSizeByName :one
SELECT *
FROM MASTER_SIZE
WHERE LOWER(BTRIM(nama_size)) = LOWER(BTRIM($1))
LIMIT 1;

-- name: ListSizes :many
SELECT *
FROM MASTER_SIZE
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_size ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_size' AND NOT sqlc.arg(sort_desc)::bool THEN id_size END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_size' AND sqlc.arg(sort_desc)::bool THEN id_size END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_size' AND NOT sqlc.arg(sort_desc)::bool THEN nama_size END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_size' AND sqlc.arg(sort_desc)::bool THEN nama_size END DESC,
    nama_size ASC,
    id_size ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountSizes :one
SELECT COUNT(*)
FROM MASTER_SIZE
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_size ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: CreateSize :one
INSERT INTO MASTER_SIZE (nama_size)
VALUES ($1)
RETURNING *;

-- name: UpdateSize :one
UPDATE MASTER_SIZE
SET nama_size = $2
WHERE id_size = $1
RETURNING *;

-- name: DeleteSize :execrows
DELETE FROM MASTER_SIZE
WHERE id_size = $1;
